// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	debug bool
)

const globalUsageMessage = `Welcome to Scanner - Azure Container Registry's dependency scanner.

To start working with Scanner, run scanner --help
`

// Execute executes the root command.
func Execute() {
	cmd := newRootCmd(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(args []string) *cobra.Command {

	cmd := &cobra.Command{
		Use:          "scanner",
		Short:        "The dependency scanner for Azure Container Registry (ACR)",
		Long:         globalUsageMessage,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose output for debugging")
	flags := cmd.PersistentFlags()

	out := cmd.OutOrStdout()

	cmd.AddCommand(
		newScanCmd(out),
		newVersionCmd(out),
	)

	_ = flags.Parse(args)

	return cmd
}
