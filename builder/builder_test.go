// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"testing"

	"github.com/Azure/acr-builder/pkg/image"
)

var (
	acb = `["testing.azurecr-test.io/testing@sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d",` +
		`"acrimageshub.azurecr.io/public/acr/acb@sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d",` +
		`"mcr.microsoft.com/acr/acb@sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d"]`
)

func TestGetRepoDigest(t *testing.T) {
	tests := []struct {
		id       int
		json     string
		imgRef   *image.Reference
		expected string
	}{
		{
			1,
			acb,
			&image.Reference{
				Registry:   "testing.azurecr-test.io",
				Repository: "testing",
			},
			"sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d",
		},
		{
			2,
			acb,
			&image.Reference{
				Registry:   "acrimageshub.azurecr.io",
				Repository: "public/acr/acb",
			},
			"sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d",
		},
		{
			3,
			acb,
			&image.Reference{
				Registry:   "mcr.microsoft.com",
				Repository: "acr/acb",
			},
			"sha256:69d6b9a450c69bde2005885fb4f850ded96596b9dd1949f4313b376e7518841d",
		},
		{
			4,
			acb,
			&image.Reference{
				Registry:   "invalid",
				Repository: "invalid",
			},
			"",
		},
	}
	for _, test := range tests {
		if actual := getRepoDigest(test.json, test.imgRef); actual != test.expected {
			t.Errorf("invalid repo digest, test id: %d; expected %s but got %s", test.id, test.expected, actual)
		}
	}
}
