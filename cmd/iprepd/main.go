package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.mozilla.org/iprepd"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "iprepd",
		Usage: "Start iprepd server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
				Value:   "./iprepd.yaml",
			},
		},
		Action: func(c *cli.Context) error {
			iprepd.StartDaemon(c.String("config"))
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "gcs-sync",
				Usage: "Create blocklist file and upload to GCS",
				Action: func(c *cli.Context) error {
					config, err := iprepd.LoadCfg(c.String("config"))
					if err != nil {
						return err
					}
					iprepd.CreateServerRuntime(c.String("config"))
					reputation, err := iprepd.RepDump()
					if err != nil {
						return err
					}
					return IPBlocklistGCS(config, reputation)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func IPBlocklistGCS(config iprepd.ServerCfg, reputationDump []iprepd.Reputation) error {
	var (
		blocklistFile       = "./ip-blocklist"
		ipBlocklistContents string
		cnt                 int
	)

	for _, rep := range reputationDump {
		if rep.Type != "ip" {
			continue
		}
		if config.Sync.MinimumReputation < rep.Reputation {
			continue
		}

		ipBlocklistContents = ipBlocklistContents + fmt.Sprintf("%s/32\n", rep.Object)

		cnt++
		if cnt == config.Sync.MaxLimit {
			break
		}

	}

	err := os.WriteFile(blocklistFile, []byte(ipBlocklistContents), 0644)
	if err != nil {
		return err
	}

	// Write to GCS
	err = uploadFileToGCS(config, blocklistFile)
	if err != nil {
		return err
	}

	// Delete based on config value
	if config.Sync.DeleteFile {
		err = os.Remove(blocklistFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadFileToGCS(config iprepd.ServerCfg, filename string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	// Open local file.
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Upload an object with storage.Writer.
	wc := client.Bucket(config.Sync.GCS.Bucketname).Object(config.Sync.GCS.Filename).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	// Make the object public
	obj := client.Bucket(config.Sync.GCS.Bucketname).Object(config.Sync.GCS.Filename)
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return err
	}

	return nil
}
