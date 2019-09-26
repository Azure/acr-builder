// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azurekeyvault

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/acr-builder/pkg/tokenutil"
	"github.com/Azure/acr-builder/vault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"
)

// AKVSecretOptions provides the options to get secret from Azure keyvault using MSI.
type AKVSecretOptions struct {
	VaultURL    string
	MSIClientID string
}

// AKVSecretFetcher provides the options to get secret from Azure keyvault using MSI.
type AKVSecretFetcher struct {
	VaultURL       string
	SecretName     string
	SecretVersion  string
	MSIClientID    string
	AADResourceURL string
}

var _ vault.SecretFetcher = &AKVSecretFetcher{}

// FetchSecretValue gets the secret vault as defined by the config from Azure key vault using MSI.
func (fetcher *AKVSecretFetcher) FetchSecretValue(ctx context.Context) (string, error) {
	if fetcher == nil {
		return "", errors.New("secret config is required")
	}

	if fetcher.VaultURL == "" ||
		fetcher.SecretName == "" ||
		fetcher.AADResourceURL == "" {
		return "", errors.New("missing required properties VaultURL, SecretName, and AADResourceURL")
	}

	keyClient, err := newKeyVaultClient(fetcher.VaultURL, fetcher.MSIClientID, fetcher.AADResourceURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to create azure key vault client")
	}

	secretValue, err := keyClient.getSecret(ctx, fetcher.SecretName, fetcher.SecretVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch secret value from azure key vault client")
	}

	return secretValue, nil
}

// NewAKVSecretConfig creates the Azure Key Vault config.
func NewAKVSecretConfig(opts *AKVSecretOptions) (vault.SecretFetcher, error) {
	if opts.VaultURL == "" {
		return nil, errors.New("missing azure keyvault URL")
	}

	normalizedVaultURL := strings.TrimSuffix(strings.ToLower(opts.VaultURL), "/")

	parsedURL, err := url.Parse(normalizedVaultURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse the azure keyvault secret URL")
	}

	if parsedURL.Scheme != "https" {
		return nil, errors.New("invalid azure keyvault secret URL scheme. Expected Https")
	}

	urlSegments := strings.Split(parsedURL.Path, "/")

	if len(urlSegments) != 3 && len(urlSegments) != 4 {
		return nil, fmt.Errorf("invalid azure keyvault secret URL. Bad number of URL segments: %d", len(urlSegments))
	}

	if urlSegments[1] != "secrets" {
		return nil, fmt.Errorf("invalid azure keyvault secret URL. Expected 'secrets' collection, but found: %s", urlSegments[1])
	}

	secretVersion := ""

	if len(urlSegments) == 4 {
		secretVersion = urlSegments[3]
	}

	vaultHostWithScheme := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	splitStr := strings.SplitAfterN(vaultHostWithScheme, ".", 2)
	// Ex. https://myacbvault.vault.azure.net -> ["https://myacbvault." "vault.azure.net"]
	if len(splitStr) != 2 {
		return nil, fmt.Errorf("extracted vault resource %s from vault URL %s is invalid", vaultHostWithScheme, opts.VaultURL)
	}

	// Ex. https://vault.azure.net
	vaultAADResourceURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, splitStr[1])

	return &AKVSecretFetcher{
		VaultURL:       vaultHostWithScheme,
		SecretName:     urlSegments[2],
		SecretVersion:  secretVersion,
		MSIClientID:    opts.MSIClientID,
		AADResourceURL: vaultAADResourceURL,
	}, nil
}

// keyVault holds the information for a keyvault instance
type keyVault struct {
	client   *keyvault.BaseClient
	vaultURL string
}

// newKeyVaultClient creates a new keyvault client
func newKeyVaultClient(vaultURL, clientID, vaultAADResourceURL string) (*keyVault, error) {
	spToken, err := tokenutil.GetServicePrincipalToken(vaultAADResourceURL, clientID)
	if err != nil {
		return nil, err
	}
	authorizer := autorest.NewBearerAuthorizer(spToken)
	keyClient := keyvault.New()
	keyClient.Authorizer = authorizer

	k := &keyVault{
		vaultURL: vaultURL,
		client:   &keyClient,
	}

	return k, nil
}

// getSecret retrieves a secret from keyvault
func (k *keyVault) getSecret(ctx context.Context, secretName, secretVersion string) (string, error) {
	secretBundle, err := k.client.GetSecret(ctx, k.vaultURL, secretName, secretVersion)
	if err != nil {
		return "", err
	}

	return *secretBundle.Value, nil
}
