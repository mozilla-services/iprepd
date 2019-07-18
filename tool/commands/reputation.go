package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"go.mozilla.org/iprepd"
	cli "gopkg.in/urfave/cli.v1"
)

// ReputationCmd is the CLI command object for reputation operations
var ReputationCmd = cli.Command{
	Name:    "reputation",
	Aliases: []string{"r"},
	Usage:   "reputation entry related commands",
	Subcommands: []cli.Command{
		{
			Name:  "list",
			Usage: "list all available reputation entries",
			Flags: []cli.Flag{
				jsonFlag,
			},
			Action: reputationListHandler,
		},
		{
			Name:  "get",
			Usage: "get the entry for a given object",
			Flags: []cli.Flag{
				asMandatory(objectFlag),
				withDefault(typeFlag, "ip"),
				jsonFlag,
			},
			Before: reputationGetValidator,
			Action: reputationGetHandler,
		},
		{
			Name:  "set",
			Usage: "update the entry for a given object",
			Flags: []cli.Flag{
				asMandatory(objectFlag),
				asMandatoryInt(scoreFlag),
				withDefault(typeFlag, "ip"),
				decayAfterFlag,
			},
			Before: reputationSetValidator,
			Action: reputationSetHandler,
		},
		{
			Name:  "clear",
			Usage: "delete the entry for a given object",
			Flags: []cli.Flag{
				asMandatory(objectFlag),
				withDefault(typeFlag, "ip"),
			},
			Before: reputationClearValidator,
			Action: reputationClearHandler,
		},
		{
			Name:  "batch-clear",
			Usage: "delete the entries for all objects in a given file",
			Flags: []cli.Flag{
				asMandatory(payloadFlag),
				withDefault(typeFlag, "ip"),
				exitOnFailFlag,
			},
			Before: reputationBatchClearValidator,
			Action: reputationBatchClearHandler,
		},
	},
}

func reputationGetValidator(ctx *cli.Context) error {
	return assertSet(ctx, objectFlag)
}

func reputationSetValidator(ctx *cli.Context) error {
	return assertSet(ctx, objectFlag, scoreFlag)
}

func reputationClearValidator(ctx *cli.Context) error {
	return assertSet(ctx, objectFlag)
}

func reputationBatchClearValidator(ctx *cli.Context) error {
	return assertSet(ctx, payloadFlag)
}

func reputationListHandler(ctx *cli.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}

	entries, err := client.Dump()
	if err != nil {
		return fmt.Errorf("could not retrieve reputation entries: %s", err)
	}

	if ctx.BoolT(name(jsonFlag)) {
		if len(entries) == 0 {
			// ensure array format, i.e. ensure we dont print "nil"
			fmt.Println("[]")
			return nil
		}
		raw, err := json.Marshal(entries)
		if err != nil {
			return fmt.Errorf("could not format response payload: %s", err)
		}
		fmt.Println(string(raw))
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("-- no entries to show --")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TYPE", "OBJECT", "SCORE"})
	for _, entry := range entries {
		if entry.Object == "" {
			if entry.IP != "" {
				table.Append([]string{"ip (legacy entry)", entry.IP, strconv.Itoa(entry.Reputation)})
			}
			continue
		}
		table.Append([]string{entry.Type, entry.Object, strconv.Itoa(entry.Reputation)})
	}
	table.Render()

	return nil
}

func reputationGetHandler(ctx *cli.Context) error {
	typ := ctx.String(name(typeFlag))
	obj := ctx.String(name(objectFlag))

	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	rept, err := client.GetReputation(typ, obj)
	if err != nil {
		return fmt.Errorf("could not get reputation for %s %s", typ, obj)
	}
	if ctx.BoolT(name(jsonFlag)) {
		raw, err := json.Marshal(rept)
		if err != nil {
			return fmt.Errorf("could not format response payload: %s", err)
		}
		fmt.Println(string(raw))
		return nil
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"OBJECT", rept.Object})
	table.Append([]string{"REPUTATION", strconv.Itoa(rept.Reputation)})
	table.Append([]string{"TYPE", rept.Type})
	table.Append([]string{"REVIEWED", strconv.FormatBool(rept.Reviewed)})
	table.Append([]string{"LAST UPDATED", rept.LastUpdated.String()})
	table.Append([]string{"DECAY AFTER", rept.DecayAfter.String()})
	table.Render()
	return nil
}

func reputationSetHandler(ctx *cli.Context) error {
	typ := ctx.String(name(typeFlag))
	obj := ctx.String(name(objectFlag))
	rep := ctx.Int(name(scoreFlag))
	da := ctx.Int(name(decayAfterFlag))

	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	if err := client.SetReputation(&iprepd.Reputation{
		Type:       typ,
		Object:     obj,
		Reputation: rep,
		DecayAfter: time.Now().Add(time.Second * time.Duration(da)),
	}); err != nil {
		return fmt.Errorf("could not update reputation for %s %s: %s", typ, obj, err)
	}
	fmt.Printf("reputation for %s %s updated successfully!\n", typ, obj)
	return nil
}

func reputationClearHandler(ctx *cli.Context) error {
	typ := ctx.String(name(typeFlag))
	obj := ctx.String(name(objectFlag))
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	if err := client.DeleteReputation(typ, obj); err != nil {
		return fmt.Errorf("could not delete reputation for %s %s: %s", typ, obj, err)
	}
	fmt.Printf("reputation for %s %s deleted successfully!\n", typ, obj)
	return nil
}

func reputationBatchClearHandler(ctx *cli.Context) error {
	typ := ctx.String(name(typeFlag))
	path := ctx.String(name(payloadFlag))
	e := ctx.Bool(name(exitOnFailFlag))

	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}

	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("could not open payload file %s: %s", path, err)
	}

	objects := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		obj := scanner.Text()
		if err := client.DeleteReputation(typ, obj); err != nil {
			if e {
				return fmt.Errorf("could not delete reputation for %s %s: %s", typ, obj, err)
			}
			continue
		}
		objects++
	}

	fmt.Printf("%d reputation entries deleted!\n", objects)
	return nil
}
