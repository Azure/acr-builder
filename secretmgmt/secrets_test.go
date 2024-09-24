// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package secretmgmt

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
)

// MockResolveSecret will mock the azure keyvault resolve and return the concatenated key vault and client ID as the value. This is used for testing purposes only.
func MockResolveSecret(_ context.Context, secret *Secret, errorChan chan error) {
	if secret == nil {
		errorChan <- errors.New("secret cannot be nil")
		return
	}

	if secret.IsKeyVaultSecret() {
		secret.ResolvedValue = fmt.Sprintf("vault-%s-%s", secret.KeyVault, secret.MsiClientID)
		secret.ResolvedChan <- true
		return
	} else if secret.IsMsiSecret() {
		secret.ResolvedValue = fmt.Sprintf("msi-%s-%s", secret.AadResourceID, secret.MsiClientID)
		secret.ResolvedChan <- true
		return
	}

	errorChan <- fmt.Errorf("cannot resolve secret with ID: %s", secret.ID)
}

// TestResolveSecrets tests resolving the secrets
func TestResolveSecrets(t *testing.T) {
	secretResolver, err := NewSecretResolver(MockResolveSecret, time.Minute*5)
	if err != nil {
		t.Errorf("Failed to create secret resolver. Err: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		secrets         []*Secret
		resolvedSecrets map[string]string
	}{
		{
			[]*Secret{
				{
					ID:       "mysecret",
					KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
				},
				{
					ID:          "mysecret1",
					KeyVault:    "https://myvault.vault.azure.net/secrets/mysecret1",
					MsiClientID: "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86",
				},
			},
			map[string]string{"mysecret": "vault-https://myvault.vault.azure.net/secrets/mysecret-", "mysecret1": "vault-https://myvault.vault.azure.net/secrets/mysecret1-c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
		},
		{
			nil,
			map[string]string{},
		},
		{
			[]*Secret{},
			map[string]string{},
		},
		{
			// Add more than 5 secrets to test the batching logic
			[]*Secret{
				{
					ID:       "1",
					KeyVault: "k1",
				},
				{
					ID:       "2",
					KeyVault: "k2",
				},
				{
					ID:       "3",
					KeyVault: "k3",
				},
				{
					ID:       "4",
					KeyVault: "k4",
				},
				{
					ID:            "5",
					AadResourceID: "k5",
				},
				{
					ID:       "6",
					KeyVault: "k6",
				},
				{
					ID:            "7",
					AadResourceID: "k7",
				},
				{
					ID:       "8",
					KeyVault: "k8",
				},
				{
					ID:            "9",
					AadResourceID: "k9",
				},
				{
					ID:       "10",
					KeyVault: "k10",
				},
			},
			map[string]string{"1": "vault-k1-", "2": "vault-k2-", "3": "vault-k3-", "4": "vault-k4-", "5": "msi-k5-", "6": "vault-k6-", "7": "msi-k7-", "8": "vault-k8-", "9": "msi-k9-", "10": "vault-k10-"},
		},
	}

	for _, test := range tests {
		err := secretResolver.ResolveSecrets(ctx, test.secrets)
		if err != nil {
			t.Errorf("Test failed with error %v", err)
		}

		for _, secret := range test.secrets {
			if val, ok := test.resolvedSecrets[secret.ID]; ok {
				actual := secret.ResolvedValue
				expected := val
				if actual != expected {
					t.Errorf("Secrets do not match. Expected  %v but got %v", expected, actual)
				}
			}
		}
	}
}

// TestResolveSecretsWithError tests resolving the secrets that should result in errors
func TestResolveSecretsWithError(t *testing.T) {
	secretResolver, err := NewSecretResolver(nil, time.Minute*5)
	if err != nil {
		t.Errorf("Failed to create secret resolver. Err: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		secrets []*Secret
	}{
		{
			[]*Secret{
				{
					ID: "mysecret",
				},
			},
		},
		{
			[]*Secret{
				{
					ID:       "mysecret",
					KeyVault: "some invalid URL",
				},
			},
		},
	}

	for _, test := range tests {
		err := secretResolver.ResolveSecrets(ctx, test.secrets)
		if err == nil {
			t.Fatalf("Expected secrets: %v to error but it didn't", test.secrets)
		}
	}
}

// TestResolveSecretsWithTimeout tests resolving the secrets with timeout should exit with error
func TestResolveSecretsWithTimeout(t *testing.T) {
	secretResolver, err := NewSecretResolver(MockResolveSecret, time.Duration(0))
	if err != nil {
		t.Errorf("Failed to create secret resolver. Err: %v", err)
	}

	ctx := context.Background()
	secrets := []*Secret{
		{
			ID:       "mysecret1",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret2",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret3",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret4",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret5",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret6",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:       "mysecret7",
			KeyVault: "https://myvault.vault.azure.net/secrets/mysecret",
		},
	}

	resolveError := secretResolver.ResolveSecrets(ctx, secrets)
	if resolveError == nil {
		t.Fatalf("Expected test to error but it didn't")
	}
}
