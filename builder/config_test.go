// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import "testing"

func TestGetRawValue_NilConfig(t *testing.T) {
	var c *Config
	expected := ""
	if actual := c.GetRawValue(); actual != expected {
		t.Fatalf("expected %s but got %s", expected, actual)
	}
}
