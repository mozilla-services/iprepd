package commands

import (
	"fmt"
	"go.mozilla.org/iprepd"
	"strings"

	cli "gopkg.in/urfave/cli.v1"
)

const (
	mandatoryTag = "[mandatory]"

	payloadFormatJSON = "json"
	payloadFormatList = "list"
)

var (
	// config flags
	pathFlag = cli.StringFlag{
		Name:  "path, p",
		Usage: "override default config file path",
	}
	urlFlag = cli.StringFlag{
		Name:  "url, u",
		Usage: "host URL to use",
	}
	tokenFlag = cli.StringFlag{
		Name:  "token, t",
		Usage: "auth token to use",
	}

	// option flags
	jsonFlag = cli.BoolFlag{
		Name:  "json, j",
		Usage: "print raw json -- don't pretty print",
	}
	exitOnFailFlag = cli.BoolFlag{
		Name:  "exit-on-fail, e",
		Usage: "exit on first encountered failure",
	}

	// input flags
	objectFlag = cli.StringFlag{
		Name:  "object, o",
		Usage: "the target object for the operation",
	}
	typeFlag = cli.StringFlag{
		Name:  "type, t",
		Usage: fmt.Sprintf("type of object e.g. \"%s\" or \"%s\"", iprepd.TypeIP, iprepd.TypeEmail),
	}
	suppressRecoveryFlag = cli.IntFlag{
		Name:  "suppress-recovery, s",
		Usage: "seconds before object's reputation begins to heal",
	}
	scoreFlag = cli.IntFlag{
		Name:  "score, s",
		Usage: "reputation score to assign",
	}
	payloadFlag = cli.StringFlag{
		Name:  "payload, p",
		Usage: "path to payload file",
	}
	payloadFmtFlag = cli.StringFlag{
		Name:  "payload-fmt, f",
		Usage: fmt.Sprintf("format of payload file provided e.g. \"%s\" or \"%s\"", payloadFormatJSON, payloadFormatList),
	}
	violationFlag = cli.StringFlag{
		Name:  "violation, v",
		Usage: "name of violation to be applied",
	}
	decayAfterFlag = cli.IntFlag{
		Name:  "decay-after, d",
		Usage: "seconds after which reputation should begin to recover",
	}
)

// name returns the long name of a flag
// note that the split function returns the original string in index 0
// if it does not contain the given delimiter ","
func name(f cli.Flag) string {
	return strings.Split(f.GetName(), ",")[0]
}

func withDefault(f cli.StringFlag, def string) cli.StringFlag {
	f.Value = def
	return f
}

func withDefaultInt(f cli.IntFlag, def int) cli.IntFlag {
	f.Value = def
	return f
}

func asMandatory(f cli.StringFlag) cli.StringFlag {
	f.Usage = fmt.Sprintf("%s %s", mandatoryTag, f.Usage)
	return f
}

func asMandatoryInt(f cli.IntFlag) cli.IntFlag {
	f.Usage = fmt.Sprintf("%s %s", mandatoryTag, f.Usage)
	return f
}

func asMandatoryIf(f cli.StringFlag, cond string) cli.StringFlag {
	f.Usage = fmt.Sprintf("[mandatory if %s] %s", cond, f.Usage)
	return f
}

func assertSet(ctx *cli.Context, flags ...cli.Flag) error {
	for _, f := range flags {
		if !ctx.IsSet(name(f)) {
			return fmt.Errorf("missing %s argument \"%s\"", mandatoryTag, name(f))
		}
	}
	return nil
}

func assertSetIf(ctx *cli.Context, cond func() bool, flags ...cli.Flag) error {
	if !cond() {
		return nil
	}
	return assertSet(ctx, flags...)
}
