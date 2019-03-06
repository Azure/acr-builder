// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/pkg/errors"
)

// MockResolveSecret will mock the azure keyvault resolve and return the concatenated Akv and client ID as the value. This is used for testing purposes only.
func MockResolveSecret(ctx context.Context, secret *graph.Secret, errorChan chan error) {
	if secret == nil {
		errorChan <- errors.New("secret cannot be nil")
		return
	}

	if secret.IsAkvSecret() {
		secret.ResolvedChan <- graph.SecretValue{ID: secret.ID, Value: fmt.Sprintf("%s-%s", secret.Akv, secret.MsiClientID)}
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
		secrets         []*graph.Secret
		resolvedSecrets Values
	}{
		{
			[]*graph.Secret{
				{
					ID:  "mysecret",
					Akv: "https://myvault.vault.azure.net/secrets/mysecret",
				},
				{
					ID:          "mysecret1",
					Akv:         "https://myvault.vault.azure.net/secrets/mysecret1",
					MsiClientID: "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86",
				},
			},
			Values{"mysecret": "https://myvault.vault.azure.net/secrets/mysecret-", "mysecret1": "https://myvault.vault.azure.net/secrets/mysecret1-c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
		},
		{
			nil,
			Values{},
		},
		{
			[]*graph.Secret{},
			Values{},
		},
		{
			// Add more than 5 secrets to test the batching logic
			[]*graph.Secret{
				{
					ID:  "1",
					Akv: "k1",
				},
				{
					ID:  "2",
					Akv: "k2",
				},
				{
					ID:  "3",
					Akv: "k3",
				},
				{
					ID:  "4",
					Akv: "k4",
				},
				{
					ID:  "5",
					Akv: "k5",
				},
				{
					ID:  "6",
					Akv: "k6",
				},
				{
					ID:  "7",
					Akv: "k7",
				},
				{
					ID:  "8",
					Akv: "k8",
				},
				{
					ID:  "9",
					Akv: "k9",
				},
				{
					ID:  "10",
					Akv: "k10",
				},
			},
			Values{"1": "k1-", "2": "k2-", "3": "k3-", "4": "k4-", "5": "k5-", "6": "k6-", "7": "k7-", "8": "k8-", "9": "k9-", "10": "k10-"},
		},
	}

	for _, test := range tests {
		resolvedSecrets, err := secretResolver.ResolveSecrets(ctx, test.secrets)
		if err != nil {
			t.Errorf("Test failed with error %v", err)
		}

		if test.resolvedSecrets == nil {
			if resolvedSecrets != nil {
				t.Errorf("Secrets do not match. Expected  %v but got %v", test.resolvedSecrets, resolvedSecrets)
			}
		} else {

			if len(resolvedSecrets) != len(test.resolvedSecrets) {
				t.Errorf("Expected number of secrets: %v, but got %v", len(test.resolvedSecrets), len(resolvedSecrets))
			}
			for key, value := range test.resolvedSecrets {
				if resolvedSecrets[key] != value {
					t.Errorf("Secrets donot match. Expected  %v but got %v", test.resolvedSecrets, resolvedSecrets)
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
		secrets []*graph.Secret
	}{
		{
			[]*graph.Secret{
				{
					ID: "mysecret",
				},
			},
		},
		{
			[]*graph.Secret{
				{
					ID:  "mysecret",
					Akv: "some invalid akv URL",
				},
			},
		},
	}

	for _, test := range tests {
		_, err := secretResolver.ResolveSecrets(ctx, test.secrets)
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
	secrets := []*graph.Secret{
		{
			ID:  "mysecret1",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret2",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret3",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret4",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret5",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret6",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
		{
			ID:  "mysecret7",
			Akv: "https://myvault.vault.azure.net/secrets/mysecret",
		},
	}

	_, resolveError := secretResolver.ResolveSecrets(ctx, secrets)
	if resolveError == nil {
		t.Fatalf("Expected test to error but it didn't")
	}
}
