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
