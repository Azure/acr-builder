// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package secretmgmt

import (
	"testing"
)

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		secret      *Secret
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&Secret{},
			true,
		},
		{
			// No vault properties
			&Secret{
				ID: "a",
			},
			true,
		},
		{
			// ID cannot contain spaces.
			&Secret{
				ID: "my secret",
			},
			true,
		},
		{
			&Secret{
				ID:  "a",
				Akv: "b",
			},
			false,
		},
		{
			// Invalid UUID for MSI Client ID
			&Secret{
				ID:          "a",
				Akv:         "b",
				MsiClientID: "test",
			},
			true,
		},
		{
			&Secret{
				ID:          "a",
				Akv:         "b",
				MsiClientID: "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86",
			},
			false,
		},
	}

	for _, test := range tests {
		err := test.secret.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected secret: %v to error but it didn't", test.secret)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("secret: %v shouldn't have errored, but it did; err: %v", test.secret, err)
		}
	}
}

func TestIsAkvSecret(t *testing.T) {
	tests := []struct {
		secret   *Secret
		expected bool
	}{
		{
			nil,
			false,
		},
		{
			&Secret{
				Akv: "a",
			},
			true,
		},
		{
			&Secret{},
			false,
		},
	}

	for _, test := range tests {
		if actual := test.secret.IsAkvSecret(); actual != test.expected {
			t.Errorf("Expected %v for secret to be AKV type, but got %v", test.expected, actual)
		}
	}
}

func TestIsMsiSecret(t *testing.T) {
	tests := []struct {
		secret   *Secret
		expected bool
	}{
		{
			nil,
			false,
		},
		{
			&Secret{
				ArmResourceID: "foo",
			},
			true,
		},
		{
			&Secret{},
			false,
		},
	}

	for _, test := range tests {
		if actual := test.secret.IsMsiSecret(); actual != test.expected {
			t.Errorf("expected %v for secret to be MSI type, but got %v", test.expected, actual)
		}
	}
}

func TestSecretEquals(t *testing.T) {
	tests := []struct {
		s        *Secret
		t        *Secret
		expected bool
	}{
		{
			nil,
			nil,
			true,
		},
		{
			&Secret{},
			&Secret{},
			true,
		},
		{
			&Secret{
				ID: "a",
			},
			nil,
			false,
		},
		{
			nil,
			&Secret{
				ID: "a",
			},
			false,
		},
		{
			&Secret{
				ID:          "a",
				Akv:         "b",
				MsiClientID: "c",
			},
			&Secret{
				ID:          "a",
				Akv:         "b",
				MsiClientID: "c",
			},
			true,
		},
	}

	for _, test := range tests {
		if actual := test.s.Equals(test.t); actual != test.expected {
			t.Errorf("Expected %v and %v to be equal to %v but got %v", test.s, test.t, test.expected, actual)
		}
	}
}
