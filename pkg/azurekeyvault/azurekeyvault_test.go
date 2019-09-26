// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azurekeyvault

import (
	"strings"
	"testing"
)

func TestNewAKVSecretConfig(t *testing.T) {
	tests := []struct {
		vaultURL             string
		shouldError          bool
		expectedSecretConfig *AKVSecretFetcher
	}{
		{
			"",
			true,
			nil,
		},
		{
			"https://tee",
			true,
			nil,
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/mysecretversion/latest",
			true,
			nil,
		},
		{
			"https://test.vault.azure.net/secrets/mysecret//mysecretversion",
			true,
			nil,
		},
		{
			"tcp://test.vault.azure.net/secrets/mysecret/mysecretversion",
			true,
			nil,
		},
		{
			"/secrets/mysecret/mysecretversion",
			true,
			nil,
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/mysecretversion",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				SecretVersion:  "mysecretversion",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/mysecretversion/",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				SecretVersion:  "mysecretversion",
				MSIClientID:    "myclientID",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault.azure.net/secrets/mysecret",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				MSIClientID:    "myclientId",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault-int.azure-int.net/secrets/mysecret/",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault-int.azure-int.net",
				SecretName:     "mysecret",
				MSIClientID:    "myclientId",
				AADResourceURL: "https://vault-int.azure-int.net",
			},
		},
		{
			"https://test.vault.azure.cn/secrets/mysecret/",
			false,
			&AKVSecretFetcher{
				VaultURL:       "https://test.vault.azure.cn",
				SecretName:     "mysecret",
				MSIClientID:    "myclientId",
				AADResourceURL: "https://vault.azure.cn",
			},
		},
	}

	for _, test := range tests {
		clientID := ""
		if test.expectedSecretConfig != nil {
			clientID = test.expectedSecretConfig.MSIClientID
		}
		fetcher, err := NewAKVSecretConfig(&AKVSecretOptions{
			VaultURL:    test.vaultURL,
			MSIClientID: clientID,
		})
		if test.shouldError && err == nil {
			t.Fatalf("Expected vaultURL: %s to error but it didn't", test.vaultURL)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("vaultURL: %s shouldn't have errored, but it did; err: %v", test.vaultURL, err)
		}

		if test.expectedSecretConfig != nil {
			if !strings.EqualFold(fetcher.(*AKVSecretFetcher).VaultURL, test.expectedSecretConfig.VaultURL) ||
				!strings.EqualFold(fetcher.(*AKVSecretFetcher).SecretName, test.expectedSecretConfig.SecretName) ||
				!strings.EqualFold(fetcher.(*AKVSecretFetcher).SecretVersion, test.expectedSecretConfig.SecretVersion) ||
				!strings.EqualFold(fetcher.(*AKVSecretFetcher).MSIClientID, test.expectedSecretConfig.MSIClientID) ||
				!strings.EqualFold(fetcher.(*AKVSecretFetcher).AADResourceURL, test.expectedSecretConfig.AADResourceURL) {
				t.Fatalf("The fetcher generated from vaultURL: %s doesn't match with expected, Generated: %v, Expected: %v", test.vaultURL, fetcher.(*AKVSecretFetcher), test.expectedSecretConfig)
			}
		}
	}
}
