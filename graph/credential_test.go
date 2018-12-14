// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import "testing"

func TestCreateCredentialFromString(t *testing.T) {
	tests := []struct {
		str              string
		registry         string
		registryUser     string
		registryPassword string
		shouldError      bool
	}{
		{"foo;bar;qux", "foo", "bar", "qux", false},
		{"foo;bar;qux;thing", "foo", "bar", "qux;thing", false},
		{"foo;bar;qux;th;i@ng", "foo", "bar", "qux;th;i@ng", false},
		{"foo;bar", "foo", "bar", "", true},
	}

	for _, test := range tests {
		actual, err := CreateCredentialFromString(test.str)
		if test.shouldError {
			if err == nil {
				t.Fatalf("Test should have errored out,but did not: %v", test)
			}
			continue
		}

		if err != nil {
			t.Fatalf("Unexpected error: %v", test)
		}

		if actual.RegistryName != test.registry ||
			actual.RegistryUsername != test.registryUser ||
			actual.RegistryPassword != test.registryPassword {
			t.Errorf("Expected %v but got %v", test, actual)
		}
	}
}

func TestNewCredential(t *testing.T) {
	tests := []struct {
		regName  string
		username string
		password string
		ok       bool
	}{
		{"foo", "bar", "qux", true},
		{"foo", "", "qux", false},
		{"", "blah", "", false},
	}

	for _, test := range tests {
		actual, err := NewCredential(test.regName, test.username, test.password)

		if !test.ok {
			if err == nil {
				t.Errorf("Expected the tests to return error but did not: %v", test)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error: %v", test)
		}

		if actual.RegistryName != test.regName ||
			actual.RegistryUsername != test.username ||
			actual.RegistryPassword != test.password {
			t.Errorf("Expected %v but got %v", test, actual)
		}
	}
}
