// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"testing"

	"github.com/pkg/errors"
)

func TestFormatErrorMessage(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{
			errors.New("context deadline exceeded"),
			"timed out",
		},
		{
			errors.Wrap(errors.New("context deadline exceeded"), "failed"),
			"failed: timed out",
		},
	}

	for _, test := range tests {
		actual := formatErrorMessage(test.err)
		if test.expected != actual {
			t.Fatalf("Expected '%s' but got '%s'", test.expected, actual)
		}
	}
}
