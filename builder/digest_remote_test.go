// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/pkg/image"
)

func TestGetReferencePath(t *testing.T) {
	tests := []struct {
		name     string
		ref      image.Reference
		expected string
		wantErr  bool
	}{
		{
			name:     "default tag",
			ref:      image.Reference{Registry: "myregistry.azurecr.io", Repository: "team/image"},
			expected: "myregistry.azurecr.io/team/image:latest",
		},
		{
			name:     "explicit tag and registry port",
			ref:      image.Reference{Registry: "localhost:5000", Repository: "team/image", Tag: "release"},
			expected: "localhost:5000/team/image:release",
		},
		{
			name:    "invalid tag",
			ref:     image.Reference{Registry: "myregistry.azurecr.io", Repository: "team/image", Tag: "invalid tag"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := getReferencePath(&test.ref)
			if (err != nil) != test.wantErr {
				t.Fatalf("expected error=%v, got %v", test.wantErr, err)
			}
			if actual != test.expected {
				t.Errorf("expected %q but got %q", test.expected, actual)
			}
		})
	}
}

func TestPopulateDigest(t *testing.T) {
	rd := &remoteDigest{}
	cxt := context.Background()
	imageRef := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/hello-world",
		Tag:        "latest",
		Reference:  "hello-world:latest",
	}

	err := rd.PopulateDigest(cxt, imageRef)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if imageRef.Digest == "" {
		t.Fatalf("image digest is not populated")
	}
}
