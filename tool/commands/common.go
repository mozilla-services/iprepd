package commands

import (
	"go.mozilla.org/iprepd"
	"go.mozilla.org/iprepd/tool/config"
	cli "gopkg.in/urfave/cli.v1"
)

func getClient(ctx *cli.Context) (*iprepd.Client, error) {
	cPath := ctx.GlobalString("config")
	config, err := config.GetConfig(cPath)
	if err != nil {
		return nil, err
	}
	return iprepd.NewClient(config.HostURL, config.AuthTK, nil)
}
