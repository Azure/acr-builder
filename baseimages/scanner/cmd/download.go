// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"context"
	"fmt"

	"io"
	"time"

	"github.com/Azure/acr-builder/baseimages/scanner/scan"
	"github.com/Azure/acr-builder/pkg/taskmanager"
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
		Short: "Download the specified context",
		Long:  scanLongDesc,
		RunE:  d.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("scanner download requires exactly 1 argument. See scanner download --help")
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

	tm := taskmanager.NewTaskManager(d.dryRun)

	scanner := scan.NewScanner(tm, d.context, "", d.destinationFolder, nil, nil, debug)
	workingDir, _, _, err := scanner.ObtainSourceCode(ctx, d.context)
	if err != nil {
		return err
	}

	fmt.Printf("Download complete, working directory: %s\n", workingDir)
	return nil
}
