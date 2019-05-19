// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	buildCmd "github.com/Azure/acr-builder/cmd/acb/commands/build"
	downloadCmd "github.com/Azure/acr-builder/cmd/acb/commands/download"
	execCmd "github.com/Azure/acr-builder/cmd/acb/commands/exec"
	getsecretCmd "github.com/Azure/acr-builder/cmd/acb/commands/getsecret"
	renderCmd "github.com/Azure/acr-builder/cmd/acb/commands/render"
	scanCmd "github.com/Azure/acr-builder/cmd/acb/commands/scan"
	versionCmd "github.com/Azure/acr-builder/cmd/acb/commands/version"
	"github.com/Azure/acr-builder/version"
	"github.com/urfave/cli"
)

func main() {
	app := New()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, formatErrorMessage(err))
		os.Exit(1)
	}
}

// New returns a *cli.App instance.
func New() *cli.App {
	app := cli.NewApp()
	app.Name = "acb"
	app.Usage = "run and build containers on Azure Container Registry"
	app.Version = version.Version
	app.Commands = []cli.Command{
		buildCmd.Command,
		downloadCmd.Command,
		execCmd.Command,
		renderCmd.Command,
		scanCmd.Command,
		versionCmd.Command,
		getsecretCmd.Command,
	}
	return app
}

func formatErrorMessage(err error) string {
	// replace the original error message "context deadline exceeded" with "timed out"
	return strings.ReplaceAll(err.Error(), context.DeadlineExceeded.Error(), "timed out")
}
