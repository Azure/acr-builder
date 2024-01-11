// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/pkg/image"
)

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
