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
		{`{"type":"opaque", "registry": "foo", "username": "bar", "password": "qux"}`, true, &RegistryCredential{
			Type:     Opaque,
			Name:     "foo",
			Username: "bar",
			Password: "qux",
			Identity: "",
		}},
		{`{"type":"opaque", "registry": "foo", "username": "", "password": "qux;th;i@ng"}`, false, nil},
		{`{"type":"vault", "registry": "r", "username": "some/vault/id", "password": "some/vault/id"}`, false, nil},
		{`{"type":"vault", "registry": "r", "username": "some/vault/id", "password": "some/vault/id", "identity": "identity/resource/id"}`, true, &RegistryCredential{
			Type:     Vault,
			Name:     "r",
			Username: "some/vault/id",
			Password: "some/vault/id",
			Identity: "identity/resource/id",
		}},
		{`{"type":"msi", "registry": "", "username": "blah", "password": "something"}`, false, nil},
	}

	for _, test := range tests {
		actual, err := CreateRegistryCredentialFromString(test.credential)

		if !test.ok {
			if err == nil {
				t.Errorf("Expected the tests to return error but did not: %v", test)
			}
			continue
		} else {
			expected := test.credObject
			if actual.Type != expected.Type ||
				actual.Name != expected.Name ||
				actual.Username != expected.Username ||
				actual.Password != expected.Password ||
				actual.Identity != expected.Identity {
				t.Fatalf("Expected %v but got %v", expected, actual)
			}
		}

		if err != nil {
			t.Errorf("Unexpected error: %v", test)
		}
	}
}
