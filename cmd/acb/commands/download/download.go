// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package download

import (
	gocontext "context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/scan"
	"github.com/urfave/cli"
)

type gitInfo struct {
	CommitID string `json:"commitID"`
	Branch   string `json:"branch"`
}

// Command downloads the specified context to a destination folder.
var Command = cli.Command{
	Name:      "download",
	Usage:     "download the specified context to a destination folder",
	ArgsUsage: "[path|url]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "evaluates the command, but doesn't execute it",
		},
		cli.StringFlag{
			Name:  "destination",
			Usage: "the destination folder for downloaded context",
			Value: "temp",
		},
		cli.Int64Flag{
			Name:  "timeout",
			Usage: "maximum execution time in seconds",
			Value: 60,
		},
	},
	Action: func(context *cli.Context) error {
		var (
			downloadCtx = context.Args().First()
			dryRun      = context.Bool("dry-run")
			destination = context.String("destination")
			timeout     = time.Duration(context.Int64("timeout")) * time.Second
		)

		if downloadCtx == "" {
			return errors.New("download requires context to be provided, see download --help")
		}

		ctx, cancel := gocontext.WithTimeout(gocontext.Background(), timeout)
		defer cancel()

		pm := procmanager.NewProcManager(dryRun)
		scanner, err := scan.NewScanner(pm, downloadCtx, "", destination, nil, nil)
		if err != nil {
			return err
		}
		workingDir, sha, branch, err := scanner.ObtainSourceCode(ctx, downloadCtx)
		if err != nil {
			return err
		}

		commitAndBranch, err := json.Marshal(&gitInfo{
			CommitID: sha,
			Branch:   branch,
		})
		if err != nil {
			return err
		}

		log.Printf("Download complete, working directory: %s\n", workingDir)
		log.Printf("CommitID and Branch information: %s\n", commitAndBranch)
		return nil
	},
}
