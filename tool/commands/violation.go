package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"go.mozilla.org/iprepd"
	cli "gopkg.in/urfave/cli.v1"
)

// ViolationCmd is the CLI command object for the violation operation
var ViolationCmd = cli.Command{
	Name:    "violation",
	Aliases: []string{"v"},
	Usage:   "violation related commands",
	Subcommands: []cli.Command{
		{
			Name:  "list",
			Usage: "list all available violations",
			Flags: []cli.Flag{
				jsonFlag,
			},
			Action: violationListHandler,
		},
		{
			Name:  "apply",
			Usage: "apply a violation to a single object",
			Flags: []cli.Flag{
				asMandatory(violationFlag),
				asMandatory(objectFlag),
				withDefault(typeFlag, "ip"),
				suppressRecoveryFlag,
			},
			Before: violationApplyValidator,
			Action: violationApplyHandler,
		},
		{
			Name:        "batch-apply",
			Description: "see https://github.com/mozilla-services/iprepd#put-violationstypeip for payload format",
			Usage:       "batch-apply violations in a payload file",
			Flags: []cli.Flag{
				asMandatory(payloadFlag),
				withDefault(payloadFmtFlag, payloadFormatJSON),
				asMandatoryIf(typeFlag, fmt.Sprintf("%s=%s", name(payloadFmtFlag), payloadFormatList)),
				asMandatoryIf(violationFlag, fmt.Sprintf("%s=%s", name(payloadFmtFlag), payloadFormatList)),
			},
			Before: violationBatchApplyValidator,
			Action: violationBatchApplyHandler,
		},
	},
}

func violationApplyValidator(ctx *cli.Context) error {
	return assertSet(ctx, violationFlag, objectFlag)
}

func violationBatchApplyValidator(ctx *cli.Context) error {
	if err := assertSet(ctx, payloadFlag); err != nil {
		return err
	}

	cond := func() bool {
		return ctx.String(name(payloadFmtFlag)) == payloadFormatList
	}

	if err := assertSetIf(ctx, cond, typeFlag, violationFlag); err != nil {
		return err
	}

	if _, err := readPayloadFile(
		ctx.String(name(payloadFlag)),
		ctx.String(name(payloadFmtFlag)),
		ctx.String(name(typeFlag)),
		ctx.String(name(violationFlag)),
	); err != nil {
		return fmt.Errorf("could not validate payload file: %s", err)
	}

	return nil
}

func readPayloadFile(path, format, objectType, violation string) ([]iprepd.ViolationRequest, error) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, fmt.Errorf("could not open payload file %s: %s", path, err)
	}
	var vrs []iprepd.ViolationRequest
	switch format {
	case payloadFormatJSON:
		dat, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("could not read payload file %s: %s", path, err)
		}
		if err = json.Unmarshal(dat, &vrs); err != nil {
			return nil, fmt.Errorf("could not unmarshal payload file - is it valid json?: %s", err)
		}
		return vrs, nil
	case payloadFormatList:
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			vrs = append(vrs, iprepd.ViolationRequest{
				Object:    scanner.Text(),
				Type:      objectType,
				Violation: violation,
			})
		}
		return vrs, nil
	default:
		return nil, fmt.Errorf("invalid payload format \"%s\"", format)
	}
}

func violationListHandler(ctx *cli.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	vs, err := client.GetViolations()
	if err != nil {
		return fmt.Errorf("could not retrieve available violation: %s", err)
	}
	if ctx.BoolT(name(jsonFlag)) {
		if len(vs) == 0 {
			// ensure array format, i.e. ensure we dont print "nil"
			fmt.Println("[]")
			return nil
		}
		raw, err := json.Marshal(vs)
		if err != nil {
			return fmt.Errorf("could not format response payload: %s", err)
		}
		fmt.Println(string(raw))
		return nil
	}

	if len(vs) == 0 {
		fmt.Println("-- no violations to show --")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "PENALTY", "DECREASE LIMIT"})
	for _, viol := range vs {
		table.Append([]string{viol.Name, strconv.Itoa(viol.Penalty), strconv.Itoa(viol.DecreaseLimit)})
	}
	table.Render()

	return nil
}

func violationApplyHandler(ctx *cli.Context) error {
	obj := ctx.String(name(objectFlag))
	typ := ctx.String(name(typeFlag))
	vio := ctx.String(name(violationFlag))
	sr := ctx.Int(name(suppressRecoveryFlag))

	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	if err = client.ApplyViolation(&iprepd.ViolationRequest{
		Object:           obj,
		Type:             typ,
		Violation:        vio,
		SuppressRecovery: sr,
	}); err != nil {
		return fmt.Errorf("could not apply violation: %s", err)
	}
	fmt.Printf("violation %s successfully applied to %s %s!\n", vio, typ, obj)
	return nil
}

func violationBatchApplyHandler(ctx *cli.Context) error {
	typ := ctx.String(name(typeFlag))
	path := ctx.String(name(payloadFlag))
	format := ctx.String(name(payloadFmtFlag))
	viol := ctx.String(name(violationFlag))
	violreqs, err := readPayloadFile(path, format, typ, viol)
	if err != nil {
		return fmt.Errorf("could not validate payload file: %s", err)
	}
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	err = client.BatchApplyViolation(typ, violreqs)
	if err != nil {
		return fmt.Errorf("could not batch apply violations: %s", err)
	}
	fmt.Printf("violation %s successfully applied to %s batch in %s!\n", viol, typ, path)
	return nil
}
