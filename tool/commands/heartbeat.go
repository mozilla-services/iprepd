package commands

import (
	"fmt"

	cli "gopkg.in/urfave/cli.v1"
)

// HeartbeatCmd is the CLI command object for the Heartbeat operation
var HeartbeatCmd = cli.Command{
	Name:   "heartbeat",
	Usage:  "http healthcheck endpoint check",
	Action: heartbeatHandler,
	Hidden: true,
}

func heartbeatHandler(ctx *cli.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize client: %s", err)
	}
	ok, err := client.Heartbeat()
	if err != nil {
		return fmt.Errorf("could not check heartbeat: %s", err)
	}
	if ok {
		fmt.Println("[OK] the server is reachable via HTTP")
	} else {
		fmt.Println("[PROBLEM] the server is NOT reachable via HTTP")
	}
	return nil
}
