// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{`Dockerfile'`, `Dockerfile`},
		{`"Dockerfile'`, `Dockerfile`},
		{`'"Dockerfile'"`, `Dockerfile`},
		{`"'"Dockerfile''"`, `Dockerfile`},
		{`'"'"Dockerfile''"'`, `Dockerfile`},
		{`Dockerfile''"'`, `Dockerfile`},
		{`Dockerfile`, `Dockerfile`},
		{`"Dockerfile"`, `Dockerfile`},
		{`'Dockerfile'`, `Dockerfile`},
		{`'Dockerfile"`, `Dockerfile`},
		{`'Dockerfile '`, `Dockerfile `},
		{`'   Dockerfile '`, `   Dockerfile `},
	}

	for _, test := range tests {
		if actual := TrimQuotes(test.s); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}

func TestTrimArtifactPrefix(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{"oci://myregistry.azurecr.io/hello-world", "myregistry.azurecr.io/hello-world"},
		{"OCI://myregistry.azurecr.io/hello-world", "myregistry.azurecr.io/hello-world"},
		{"OCI://", "OCI://"},
		{"myregistry.azurecr.io/hello-world", "myregistry.azurecr.io/hello-world"},
	}

	for _, test := range tests {
		if actual := TrimArtifactPrefix(test.s); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}
