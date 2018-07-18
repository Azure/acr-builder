// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	versionLongMessage = `
Prints the version information
`
)

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  versionLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf(`Go version: %s
Go compiler: %s
Platform: %s/%s
`, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}

	return cmd
}
