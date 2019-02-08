// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package getsecret

import (
	gocontext "context"
	"errors"
	"log"

	"github.com/Azure/acr-builder/vaults"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/urfave/cli"
)

// Command fetches secret from supported vaults displays the secret vaule as output.
var Command = cli.Command{
	Name:  "getsecret",
	Usage: "gets the secret value from a specified vault.",
	Subcommands: []cli.Command{
		{
			Name:  "akv",
			Usage: "gets the secret value from azure keyvault. If it is an azure keyvault secret, it is assumed that the host has the MSI token service running at http://169.254.169.254/.",
			Flags: []cli.Flag{
				// options
				cli.StringFlag{
					Name:  "url",
					Usage: "the azure keyvault secret URL",
				},
				cli.StringFlag{
					Name:  "client-id",
					Usage: "the MSI user assigned identity client ID",
				},
				cli.StringFlag{
					Name:  "vault-resource-url",
					Usage: "the resource URL for the azure key vault to get AAD token",
				},
			},
			Action: func(context *cli.Context) error {
				var (
					url                 = context.String("url")
					clientID            = context.String("client-id")
					vaultAADResourceURL = context.String("vault-resource-url")
				)

				if url == "" {
					return errors.New("secret url is required")
				}

				if vaultAADResourceURL == "" {
					vaultAADResourceURL = azure.PublicCloud.KeyVaultEndpoint
				}

				secretConfig, err := vaults.NewAKVSecretConfig(url, clientID, vaultAADResourceURL)
				if err != nil {
					return err
				}

				secretValue, err := secretConfig.GetValue(gocontext.Background())
				if err != nil {
					return err
				}
				log.Println("The secret value:")
				log.Println(secretValue)
				return nil
			},
		},
	},
}
