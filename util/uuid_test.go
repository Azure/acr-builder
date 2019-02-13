// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		uid      string
		expected bool
	}{
		{"test", false},
		{"", false},
		{"c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86", true},
	}
	for _, test := range tests {
		if actual := IsValidUUID(test.uid); actual != test.expected {
			t.Errorf("Expected result for uuid %s to be %v but got %v", test.uid, test.expected, actual)
		}
	}
}
