// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"testing"
)

// TestisAlphanumeric: Makes sure TestisAlphanumeric is performing as expected
func TestIsAlphanumeric(t *testing.T) {
	alphaNum := "abcdefghijklmnopqrstuvxyzABCDEFGHIJKLMNOPQRSTUVXZ0123456789"
	nonAlpha := "ñ-_[]{}';./'#@$%^&*()+=♕⛄ ☀üÌÅ"

	for _, char := range alphaNum {
		if isAlphanumeric(char) != true {
			t.Fatalf("TestisAlphanumeric failed + " + string(char) + "improperly classified as non alphanumeric")
		}
	}

	for _, char := range nonAlpha {
		if isAlphanumeric(char) == true {
			t.Fatalf("TestisAlphanumeric failed + " + string(char) + "improperly classified as non alphanumeric")
		}
	}
}
