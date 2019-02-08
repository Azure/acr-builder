// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package getsecret

import (
	"errors"
	"log"

	"github.com/Azure/acr-builder/vaults"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/urfave/cli"
)

// Command renders templates and verifies their output.
var Command = cli.Command{
	Name:  "getsecret",
	Usage: "gets the secret value as given by the parameters. If it is an azure keyvault secret, it is assumed that the host has the MSI token service running at http://169.254.169.254/.",
	Flags: []cli.Flag{
		// options
		cli.StringFlag{
			Name:  "akv",
			Usage: "the azure keyvault secret URL",
		},
		cli.StringFlag{
			Name:  "clientID",
			Usage: "the MSI user assigned identity client ID",
		},

		// Rendering options
		cli.StringFlag{
			Name:  "vaultAADResourceURL",
			Usage: "the resource URL for the azure key vault to get AAD token",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			akv                 = context.String("akv")
			clientID            = context.String("clientID")
			vaultAADResourceURL = context.String("vaultAADResourceURL")
		)

		if akv == "" {
			return errors.New("akv is required")
		}

		if vaultAADResourceURL == "" {
			vaultAADResourceURL = azure.PublicCloud.KeyVaultEndpoint
		}

		secretConfig, err := vaults.NewAKVSecretConfig(akv, clientID, vaultAADResourceURL)
		if err != nil {
			return err
		}

		secretValue, err := secretConfig.GetValue()
		if err != nil {
			return err
		}
		log.Println("The secret value:")
		log.Println(secretValue)
		return nil
	},
}
