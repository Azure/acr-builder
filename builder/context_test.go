// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"testing"

	"github.com/Azure/acr-builder/pkg/image"
)

func TestGetImageDependencies(t *testing.T) {
	json := `[{"image":{"registry":"registry.hub.docker.com","repository":"library/scanner","tag":"latest","digest":"","reference":"scanner:latest"},` +
		`"runtime-dependency":{"registry":"registry.hub.docker.com","repository":"library/alpine","tag":"latest","digest":"","reference":"alpine:latest"},` +
		`"buildtime-dependency":[{"registry":"registry.hub.docker.com","repository":"library/golang","tag":"1.10-alpine","digest":"test","reference":"golang:1.10-alpine"}],` +
		`"git":{"git-head-revision":"abcdef"}}]`

	deps, err := getImageDependencies(json)
	if err != nil {
		t.Fatalf("Unexpected err while parsing image deps: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}
	r := deps[0]
	if r.Image == nil {
		t.Fatal("Image shouldn't be nil")
	}
	if r.Runtime == nil {
		t.Fatal("Runtime shouldn't be nil")
	}
	if len(r.Buildtime) != 1 {
		t.Fatalf("Expected 1 buildtime dep, got %d", len(r.Buildtime))
	}
	if r.Git == nil {
		t.Fatal("git shouldn't be nil")
	}

	expectedImage := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/scanner",
		Tag:        "latest",
		Digest:     "",
		Reference:  "scanner:latest",
	}
	expectedRuntimeDep := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/alpine",
		Tag:        "latest",
		Digest:     "",
		Reference:  "alpine:latest",
	}
	expectedBuildDep := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/golang",
		Tag:        "1.10-alpine",
		Digest:     "test",
		Reference:  "golang:1.10-alpine",
	}
	expectedGitHeadRev := "abcdef"

	if !image.ReferencesEquals(r.Image, expectedImage) {
		t.Errorf("Invalid image ref. Expected %s, got %s", expectedImage.String(), r.Image.String())
	}
	if !image.ReferencesEquals(r.Runtime, expectedRuntimeDep) {
		t.Errorf("Invalid runtime dep. Expected %s, got %s", expectedRuntimeDep.String(), r.Runtime.String())
	}
	if !image.ReferencesEquals(r.Buildtime[0], expectedBuildDep) {
		t.Errorf("Invalid buildtime dep. Expected %s, got %s", expectedBuildDep.String(), r.Buildtime[0].String())
	}
	if r.Git.GitHeadRev != expectedGitHeadRev {
		t.Errorf("Invalid git head rev. Expected %s, got %s", expectedGitHeadRev, r.Git.GitHeadRev)
	}
}
