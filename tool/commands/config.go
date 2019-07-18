package commands

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"go.mozilla.org/iprepd/tool/config"
	cli "gopkg.in/urfave/cli.v1"
)

// ConfigCmd is the CLI command object for the config operation
var ConfigCmd = cli.Command{
	Name:  "config",
	Usage: "configure command line tool",
	Subcommands: []cli.Command{
		{
			Name:  "set",
			Usage: "create configuration file with given options",
			Flags: []cli.Flag{
				asMandatory(urlFlag),
				asMandatory(tokenFlag),
				pathFlag,
			},
			Before: configSetValidator,
			Action: configSetHandler,
		},
		{
			Name:  "show",
			Usage: "show contents of set configuration file",
			Flags: []cli.Flag{
				pathFlag,
			},
			Action: configShowHandler,
		},
	},
}

func configSetValidator(ctx *cli.Context) error {
	return assertSet(ctx, urlFlag, tokenFlag)
}

func configSetHandler(ctx *cli.Context) error {
	if err := config.SetConfig(
		ctx.String(name(urlFlag)),
		ctx.String(name(tokenFlag)),
		ctx.String(name(pathFlag)),
	); err != nil {
		return fmt.Errorf("could not set configuration: %s", err)
	}
	return nil
}

func configShowHandler(ctx *cli.Context) error {
	path := ctx.String(name(pathFlag))
	c, err := config.GetConfig(path)
	if err != nil {
		return fmt.Errorf("could not retrive configuration from %s: %s", path, err)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"HOST_URL", c.HostURL})
	table.Append([]string{"AUTH_TK", c.AuthTK})
	table.Render()
	return nil
}
