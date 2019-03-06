// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package vaults

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
)

const (
	// environment variable to override the default msi endpoint
	envAcrMsiEndpoint = "ACR_MSI_ENDPOINT"
)

// AKVSecretConfig provides the options to get secret from Azure keyvault using MSI.
type AKVSecretConfig struct {
	VaultURL       string
	SecretName     string
	SecretVersion  string
	MSIClientID    string
	AADResourceURL string
}

// GetValue gets the secret vaule as defined by the config from Azure key vault using MSI.
func (secretConfig *AKVSecretConfig) GetValue(ctx context.Context) (string, error) {
	if secretConfig == nil {
		return "", errors.New("secret config is required")
	}

	if secretConfig.VaultURL == "" ||
		secretConfig.SecretName == "" ||
		secretConfig.AADResourceURL == "" {
		return "", errors.New("missing required properties VaultURL, SecretName, and AADResourceURL")
	}

	keyClient, err := newKeyVaultClient(secretConfig.VaultURL, secretConfig.MSIClientID, secretConfig.AADResourceURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to create azure key vault client")
	}

	secretValue, err := keyClient.getSecret(ctx, secretConfig.SecretName, secretConfig.SecretVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch secret value from azure key vault client")
	}

	return secretValue, nil
}

// NewAKVSecretConfig creates the Azure Key Vault config.
func NewAKVSecretConfig(vaultURL, msiClientID string) (*AKVSecretConfig, error) {
	if vaultURL == "" {
		return nil, errors.New("missing azure keyvault URL")
	}

	normalizedVaultURL := strings.TrimSuffix(strings.ToLower(vaultURL), "/")

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
		return nil, fmt.Errorf("extracted vault resource %s from vault URL %s is invalid", vaultHostWithScheme, vaultURL)
	}

	// Ex. https://vault.azure.net
	vaultAADResourceURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, splitStr[1])

	akvConfig := &AKVSecretConfig{
		VaultURL:       vaultHostWithScheme,
		SecretName:     urlSegments[2],
		SecretVersion:  secretVersion,
		MSIClientID:    msiClientID,
		AADResourceURL: vaultAADResourceURL,
	}

	return akvConfig, nil
}

// keyVault holds the information for a keyvault instance
type keyVault struct {
	client   *keyvault.BaseClient
	vaultURL string
}

// newKeyVaultClient creates a new keyvault client
func newKeyVaultClient(vaultURL, clientID, vaultAADResourceURL string) (*keyVault, error) {
	msiKeyConfig := &auth.MSIConfig{
		Resource: vaultAADResourceURL,
		ClientID: clientID,
	}

	authorizer, err := newAuthorizer(msiKeyConfig)
	if err != nil {
		return nil, err
	}

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

// newAuthorizer creates the authorizer for kev vault client.
// it is based on github.com/Azure/acr-builder/vendor/github.com/Azure/go-autorest/autorest/azure/auth/auth.go and allows overriding the msi endpont using environment variable
func newAuthorizer(mc *auth.MSIConfig) (autorest.Authorizer, error) {
	// default to the well known endpoint for getting MSI authentications tokens
	msiEndpoint := "http://169.254.169.254/metadata/identity/oauth2/token"

	// override the default from environment variable
	if acrMSIEndpoint := os.Getenv(envAcrMsiEndpoint); acrMSIEndpoint != "" {
		msiEndpoint = acrMSIEndpoint
	}

	var spToken *adal.ServicePrincipalToken
	var err error
	if mc.ClientID == "" {
		spToken, err = adal.NewServicePrincipalTokenFromMSI(msiEndpoint, mc.Resource)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token from MSI: %v", err)
		}
	} else {
		spToken, err = adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, mc.Resource, mc.ClientID)
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token from MSI for user assigned identity: %v", err)
		}
	}

	return autorest.NewBearerAuthorizer(spToken), nil
}
