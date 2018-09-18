// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

func TestContainsSpace(t *testing.T) {
	tests := []struct {
		s        string
		expected bool
	}{
		{"\t", true},
		{" ", true},
		{"foo bar", true},
		{"foo   ", true},
		{"", false},
		{"foo", false},
		{"王明：这是什么？", false},
	}

	for _, test := range tests {
		if actual := ContainsSpace(test.s); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}
