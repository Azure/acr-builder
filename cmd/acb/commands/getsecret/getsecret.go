// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package getsecret

import (
	gocontext "context"
	"errors"
	"log"
	"time"

	"github.com/Azure/acr-builder/pkg/azurekeyvault"
	"github.com/urfave/cli"
)

const (
	defaultTimeoutInSeconds = 30
)

var (
	errInvalidURL = errors.New("secret url is required")
)

// Command fetches secret from supported vaults displays the secret vaule as output.
var Command = cli.Command{
	Name:  "getsecret",
	Usage: "gets the secret value from a specified vault",
	Subcommands: []cli.Command{
		{
			Name:  "keyvault",
			Usage: "gets the secret value from a key vault. If it is an Azure Key Vault (AKV) secret, it is assumed that the host has the MSI token service running at http://169.254.169.254/.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "url",
					Usage: "the secret URL",
				},
				cli.StringFlag{
					Name:  "client-id",
					Usage: "the MSI user assigned identity client ID",
				},
			},
			Action: func(context *cli.Context) error {
				var (
					url      = context.String("url")
					clientID = context.String("client-id")
				)

				if url == "" {
					return errInvalidURL
				}

				secretConfig, err := azurekeyvault.NewAKVSecretConfig(&azurekeyvault.AKVSecretOptions{
					VaultURL:    url,
					MSIClientID: clientID,
				})
				if err != nil {
					return err
				}

				timeout := time.Duration(defaultTimeoutInSeconds) * time.Second
				ctx, cancel := gocontext.WithTimeout(gocontext.Background(), timeout)
				defer cancel()

				secretValue, err := secretConfig.FetchSecretValue(ctx)
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
