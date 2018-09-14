// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

func TestIsInterfaceMap(t *testing.T) {
	tests := []struct {
		v        interface{}
		expected bool
	}{
		{"", false},
		{map[string]interface{}{}, true},
		{map[string]string{}, false},
	}

	for _, test := range tests {
		if actual := IsInterfaceMap(test.v); actual != test.expected {
			t.Errorf("Expected %v to be %v but got %v", test.v, test.expected, actual)
		}
	}
}
