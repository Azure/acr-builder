// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"fmt"
	"os"

	buildCmd "github.com/Azure/acr-builder/cmd/acb/commands/build"
	downloadCmd "github.com/Azure/acr-builder/cmd/acb/commands/download"
	execCmd "github.com/Azure/acr-builder/cmd/acb/commands/exec"
	renderCmd "github.com/Azure/acr-builder/cmd/acb/commands/render"
	scanCmd "github.com/Azure/acr-builder/cmd/acb/commands/scan"
	versionCmd "github.com/Azure/acr-builder/cmd/acb/commands/version"
	"github.com/Azure/acr-builder/version"
	"github.com/urfave/cli"
)

func main() {
	app := New()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	}
	return app
}
