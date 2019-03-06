// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package vaults

import (
	"strings"
	"testing"
)

func TestNewAKVSecretConfig(t *testing.T) {
	tests := []struct {
		vaultURL             string
		shouldError          bool
		expectedSecretConfig *AKVSecretConfig
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
			&AKVSecretConfig{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				SecretVersion:  "mysecretversion",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/mysecretversion/",
			false,
			&AKVSecretConfig{
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
			&AKVSecretConfig{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault.azure.net/secrets/mysecret/",
			false,
			&AKVSecretConfig{
				VaultURL:       "https://test.vault.azure.net",
				SecretName:     "mysecret",
				MSIClientID:    "myclientId",
				AADResourceURL: "https://vault.azure.net",
			},
		},
		{
			"https://test.vault-int.azure-int.net/secrets/mysecret/",
			false,
			&AKVSecretConfig{
				VaultURL:       "https://test.vault-int.azure-int.net",
				SecretName:     "mysecret",
				MSIClientID:    "myclientId",
				AADResourceURL: "https://vault-int.azure-int.net",
			},
		},
		{
			"https://test.vault.azure.cn/secrets/mysecret/",
			false,
			&AKVSecretConfig{
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
		config, err := NewAKVSecretConfig(test.vaultURL, clientID)
		if test.shouldError && err == nil {
			t.Fatalf("Expected vaultURL: %s to error but it didn't", test.vaultURL)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("vaultURL: %s shouldn't have errored, but it did; err: %v", test.vaultURL, err)
		}

		if test.expectedSecretConfig != nil {
			if !strings.EqualFold(config.VaultURL, test.expectedSecretConfig.VaultURL) ||
				!strings.EqualFold(config.SecretName, test.expectedSecretConfig.SecretName) ||
				!strings.EqualFold(config.SecretVersion, test.expectedSecretConfig.SecretVersion) ||
				!strings.EqualFold(config.MSIClientID, test.expectedSecretConfig.MSIClientID) ||
				!strings.EqualFold(config.AADResourceURL, test.expectedSecretConfig.AADResourceURL) {
				t.Fatalf("The config generated from vaultURL: %s doesn't match with expected, Generated: %v, Expected: %v", test.vaultURL, config, test.expectedSecretConfig)
			}
		}
	}
}
