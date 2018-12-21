// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/scan"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const downloadLongDesc = `
This command can be used to download context to a location.
`

type downloadCmd struct {
	out               io.Writer
	context           string
	timeout           int
	destinationFolder string
	dryRun            bool
}

func newDownloadCmd(out io.Writer) *cobra.Command {
	d := &downloadCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "download [OPTIONS] PATH | URL",
		Short: "Download the specified context to a destination folder",
		Long:  downloadLongDesc,
		RunE:  d.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("download requires exactly 1 argument. See acb download --help")
			}

			return nil
		},
	}

	f := cmd.Flags()

	f.BoolVar(&d.dryRun, "dry-run", false, "evaluates the command but doesn't execute it")
	f.IntVar(&d.timeout, "timeout", 60, "maximum execution time (in seconds)")
	f.StringVar(&d.destinationFolder, "destination", "temp", "the destination folder to save context")
	return cmd
}

func (d *downloadCmd) run(cmd *cobra.Command, args []string) error {
	d.context = args[0]
	timeout := time.Duration(d.timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pm := procmanager.NewProcManager(d.dryRun)

	scanner, err := scan.NewScanner(pm, d.context, "", d.destinationFolder, nil, nil, debug)
	if err != nil {
		return err
	}
	workingDir, _, err := scanner.ObtainSourceCode(ctx, d.context)
	if err != nil {
		return err
	}

	log.Printf("Download complete, working directory: %s\n", workingDir)
	return nil
}
