// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package version

import (
	"fmt"
	"runtime"

	"github.com/Azure/acr-builder/version"
	"github.com/urfave/cli"
)

// Command prints the client and runtime versions.
var Command = cli.Command{
	Name:  "version",
	Usage: "print the client and runtime versions",
	Action: func(_ *cli.Context) error {
		fmt.Println("Client:")
		fmt.Println("  Version:", version.Version)
		fmt.Println("  Revision:", version.Revision)
		fmt.Println("  Go version:", runtime.Version())
		fmt.Println("  Go compiler:", runtime.Compiler)
		fmt.Println("  Platform:", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}
