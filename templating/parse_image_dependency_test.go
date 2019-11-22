// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"testing"

	"github.com/Azure/acr-builder/util"
	"github.com/docker/distribution/reference"
)

// TestParseImageDependency validates that we can handle quotes in the Dockerfile FROM cmd directive.
func TestParseImageDependency(t *testing.T) {
	tests := []struct {
		s string
	}{
		{`alpine`},
		{`"alpine"`},
		{`'alpine'`},
	}

	for _, test := range tests {
		ref, err := reference.Parse(util.TrimQuotes(test.s))
		if err != nil && ref == nil {
			t.Errorf("Could not parse image reference %v", test.s)
		}
	}
}
