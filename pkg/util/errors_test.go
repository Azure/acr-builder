// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"errors"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		errors   Errors
		expected string
	}{
		{
			nil,
			"",
		},
		{
			Errors{
				errors.New("a"),
			},
			"a",
		},
		{
			Errors{
				errors.New("a"),
				errors.New("b"),
			},
			"a, b",
		},
	}

	for _, test := range tests {
		if actual := test.errors.String(); actual != test.expected {
			t.Errorf("expected %s but got %s", test.expected, actual)
		}
	}
}
