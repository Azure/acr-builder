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

const globalUsageMessage = `Welcome to Rally - Azure Container Registry's container builder.

To start working with Rally, run rally --help
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
		Use:          "rally",
		Short:        "The builder for Azure Container Registry (ACR)",
		Long:         globalUsageMessage,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose output for debugging")
	flags := cmd.PersistentFlags()

	out := cmd.OutOrStdout()

	cmd.AddCommand(
		newVersionCmd(out),
		newExecCmd(out),
		newBuildCmd(out),
		newLintCmd(out),
		newInitCmd(out),
	)

	_ = flags.Parse(args)

	return cmd
}
