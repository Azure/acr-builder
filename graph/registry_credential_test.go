// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import "testing"

func TestCreateCredentialFromString(t *testing.T) {
	tests := []struct {
		credential string
		ok         bool
		credObject *RegistryCredential
	}{
		{`{"usernameProviderType": "opaque","passwordProviderType":"opaque", "registry": "foo", "username": "bar", "password": "qux"}`, true, &RegistryCredential{
			Registry:     "foo",
			Username:     "bar",
			UsernameType: Opaque,
			Password:     "qux",
			PasswordType: Opaque,
			Identity:     "",
		}},
		{`{"usernameProviderType":"opaque","passwordProviderType":"opaque","registry":"foo","username":"","password":"qux;th;i@ng"}`, false, nil},
		{`{"usernameProviderType":"opaque","passwordProviderType":"vaultsecret","registry":"r","username":"my_username","password":"some/vault/id"}`, false, nil},
		{`{"usernameProviderType":"opaque","passwordProviderType":"vaultsecret","registry":"r","username":"my_username","password":"some/vault/id", "identity":"clientID"}`, true, &RegistryCredential{
			Registry:     "r",
			Username:     "my_username",
			UsernameType: Opaque,
			Password:     "some/vault/id",
			PasswordType: VaultSecret,
			Identity:     "clientID",
		}},
		{`{"registry":"r","identity":"clientID"}`, false, nil},
		{`{"registry":"r","identity":"clientID", "aadResourceId": "https://management.azure.com"}`, true, &RegistryCredential{
			Registry:      "r",
			Identity:      "clientID",
			AadResourceID: "https://management.azure.com",
		}},
		{`{"registry": "", "username": "blah", "password": "something"}`, false, nil},
	}

	for _, test := range tests {
		actual, err := CreateRegistryCredentialFromString(test.credential)

		if !test.ok {
			if err == nil {
				t.Errorf("Expected the tests to return error but did not: %v", test)
			}
			continue
		}

		expected := test.credObject
		if !actual.Equals(expected) {
			t.Fatalf("Expected %v but got %v", expected, actual)
		}

		if err != nil {
			t.Errorf("Unexpected error: %v", test)
		}
	}
}

func TestCreateCredentialFromList(t *testing.T) {
	tests := []struct {
		credential []string
		ok         bool
		credList   []*RegistryCredential
	}{
		{[]string{
			`{"usernameProviderType": "opaque","passwordProviderType":"opaque", "registry": "foo", "username": "bar", "password": "qux"}`,
			`{"usernameProviderType":"opaque","passwordProviderType":"vaultsecret","registry":"r","username":"my_username","password":"some/vault/id", "identity":"clientID"}`,
		}, true, []*RegistryCredential{
			{
				Registry:     "foo",
				Username:     "bar",
				UsernameType: Opaque,
				Password:     "qux",
				PasswordType: Opaque,
				Identity:     "",
			},
			{
				Registry:     "r",
				Username:     "my_username",
				UsernameType: Opaque,
				Password:     "some/vault/id",
				PasswordType: VaultSecret,
				Identity:     "clientID",
			},
		},
		},
	}

	for _, test := range tests {
		actual, err := CreateRegistryCredentialFromList(test.credential)

		if !test.ok {
			if err == nil {
				t.Errorf("Expected the tests to return error but did not: %v", test)
			}
			continue
		}

		for i, a := range actual {
			expected := test.credList[i]
			if !a.Equals(expected) {
				t.Fatalf("Expected %v but got %v", expected, actual)
			}
		}

		if err != nil {
			t.Errorf("Unexpected error: %v", test)
		}
	}
}
