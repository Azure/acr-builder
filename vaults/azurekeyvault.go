// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package vaults

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
func (secretConfig *AKVSecretConfig) GetValue() (string, error) {
	if secretConfig == nil {
		return "", errors.New("secret config is required")
	}

	if secretConfig.VaultURL == "" ||
		secretConfig.SecretName == "" {
		return "", errors.New("missing required properties vaultURL and SecretName")
	}

	keyClient, err := newKeyVaultClient(secretConfig.VaultURL, secretConfig.MSIClientID, secretConfig.AADResourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to create azure key vault client. Error: %v", err)
	}

	secretValue, err := keyClient.getSecret(secretConfig.SecretName, secretConfig.SecretVersion)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret value from azure key vault client. Error: %v", err)
	}

	return secretValue, nil
}

// NewAKVSecretConfig creates the Azure Key Vault config.
func NewAKVSecretConfig(vaultURL, msiClientID, vaultAADResourceURL string) (*AKVSecretConfig, error) {
	if vaultURL == "" {
		return nil, fmt.Errorf("missing azure keyvault URL")
	}

	normalizedVaultURL := strings.TrimSuffix(strings.ToLower(vaultURL), "/")

	url, err := url.Parse(normalizedVaultURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the azure keyvault secret URL. Error: %v", err)
	}

	if url.Scheme != "https" {
		return nil, fmt.Errorf("invalid azure keyvault secret URL scheme. Expected Https")
	}

	urlSegments := strings.Split(url.Path, "/")

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

	akvConfig := &AKVSecretConfig{
		VaultURL:       fmt.Sprintf("%s://%s", url.Scheme, url.Host),
		SecretName:     urlSegments[2],
		SecretVersion:  secretVersion,
		MSIClientID:    msiClientID,
		AADResourceURL: vaultAADResourceURL,
	}

	return akvConfig, nil
}

// KeyVault holds the information for a keyvault instance
type keyVault struct {
	client   *keyvault.BaseClient
	vaultURL string
}

// NewKeyVaultClient creates a new keyvault client
func newKeyVaultClient(vaultURL, clientID, vaultAADResourceURL string) (*keyVault, error) {

	if vaultAADResourceURL == "" {
		vaultAADResourceURL = azure.PublicCloud.KeyVaultEndpoint
	}

	msiKeyConfig := &auth.MSIConfig{
		Resource: strings.TrimSuffix(vaultAADResourceURL, "/"),
		ClientID: clientID,
	}

	auth, err := msiKeyConfig.Authorizer()
	if err != nil {
		return nil, err
	}

	keyClient := keyvault.New()
	keyClient.Authorizer = auth

	k := &keyVault{
		vaultURL: vaultURL,
		client:   &keyClient,
	}

	return k, nil
}

// GetSecret retrieves a secret from keyvault
func (k *keyVault) getSecret(secretName, secretVersion string) (string, error) {
	ctx := context.Background()

	secretBundle, err := k.client.GetSecret(ctx, k.vaultURL, secretName, secretVersion)
	if err != nil {
		return "", err
	}

	return *secretBundle.Value, nil
}
