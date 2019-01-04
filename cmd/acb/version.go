// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/Azure/acr-builder/version"
	"github.com/spf13/cobra"
)

const (
	versionLongMessage = `
Prints version information
`
)

func newVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  versionLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf(`Version: %s
Revision: %s
Go version: %s
Go compiler: %s
Platform: %s/%s
`, version.Version, version.Revision, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}

	return cmd
}
