// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	gocontext "context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/scan"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Command scans a Dockerfile for dependencies.
var Command = cli.Command{
	Name:      "scan",
	Usage:     "scan a Dockerfile for dependencies",
	ArgsUsage: "[path|url]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "evaluates the command, but doesn't execute it",
		},
		cli.BoolFlag{
			Name:  "cleanup",
			Usage: "delete the destination folder after execution",
		},
		cli.StringFlag{
			Name:  "file,f",
			Usage: "the path to the Dockerfile",
			Value: "Dockerfile",
		},
		cli.StringFlag{
			Name:  "destination",
			Usage: "the destination folder where downloaded context will be saved",
			Value: "temp",
		},
		cli.StringSliceFlag{
			Name:  "tag,t",
			Usage: "name and optionally a tag in 'name:tag' format",
		},
		cli.StringSliceFlag{
			Name:  "build-arg",
			Usage: "build arguments",
		},
		cli.Int64Flag{
			Name:  "timeout",
			Usage: "maximum execution time in seconds",
			Value: 60,
		},
	},
	Action: func(context *cli.Context) error {
		var (
			downloadCtx = context.Args().Get(0)
			dryRun      = context.Bool("dry-run")
			cleanup     = context.Bool("cleanup")
			dockerfile  = context.String("file")
			destination = context.String("destination")
			tags        = context.StringSlice("tag")
			buildArgs   = context.StringSlice("build-arg")
			timeout     = time.Duration(context.Int64("timeout")) * time.Second
		)

		if downloadCtx == "" {
			return errors.New("scan requires context to be provided, see scan --help")
		}

		ctx, cancel := gocontext.WithTimeout(gocontext.Background(), timeout)
		defer cancel()

		if cleanup {
			defer func() {
				_ = os.RemoveAll(destination)
			}()
		}

		pm := procmanager.NewProcManager(dryRun)
		scanner, err := scan.NewScanner(pm, downloadCtx, dockerfile, destination, buildArgs, tags)
		if err != nil {
			return err
		}

		deps, err := scanner.Scan(ctx)
		if err != nil {
			return err
		}

		bytes, err := json.Marshal(deps)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal image dependencies")
		}

		log.Println("Dependencies:")
		log.Println(string(bytes))
		return nil
	},
}
