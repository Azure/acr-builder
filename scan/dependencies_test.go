// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Azure/acr-builder/pkg/image"
)

// TestResolveDockerfileDependencies tests resolving runtime and build time dependencies from a Dockerfile.
func TestResolveDockerfileDependencies(t *testing.T) {
	ver := "2.0.6"
	buildImg := "aspnetcore-build"

	expectedRuntime := fmt.Sprintf("microsoft/aspnetcore:%s", ver)
	expectedBuildDeps := map[string]bool{
		fmt.Sprintf("microsoft/%s:%s", buildImg, ver): true,
		"imaginary/cert-generator:1.0":                true,
	}

	args := []string{fmt.Sprintf("build_image=%s", buildImg), fmt.Sprintf("build_version=%s", ver)}

	df := []byte(`# This dockerfile is just a test resource and is not buildable
ARG runtime_version="2.0.6"
FROM microsoft/aspnetcore:${runtime_version} AS base
WORKDIR /app
EXPOSE 80

FROM base AS secondary
RUN nothing

ARG build_image
ARG build_version=2.0
FROM microsoft/$build_image:$build_version AS builder
WORKDIR /src
COPY *.sln ./
COPY Web/Web.csproj Web/
RUN dotnet restore
COPY . .
WORKDIR /src/Web
RUN dotnet build -c Release -o /app

FROM builder AS publish
RUN dotnet publish -c Release -o /app

FROM imaginary/cert-generator:1.0
RUN generate-cert.sh

FROM secondary AS production
WORKDIR /app
COPY --from=publish /app .
COPY --from=3 /cert /app
ENTRYPOINT ["dotnet", "Web.dll"]`)

	runtimeDep, buildDeps, err := resolveDockerfileDependencies(bytes.NewReader(df), args, "")

	if err != nil {
		t.Errorf("Failed to resolve dependencies: %v", err)
	}

	if runtimeDep != expectedRuntime {
		t.Errorf("Unexpected runtime. Got %s, expected %s", runtimeDep, expectedRuntime)
	}

	for _, buildDep := range buildDeps {
		if ok := expectedBuildDeps[buildDep]; !ok {
			t.Errorf("Unexpected build-time dependencies. Got %v which wasn't expected", buildDep)
		}
	}
}

func TestResolveDockerfileDependencies_WithBOM(t *testing.T) {
	expectedRuntime := "scratch"
	expectedBuildDeps := map[string]bool{
		fmt.Sprintf("golang:1.10-alpine"): true,
	}
	df := []byte(`FROM golang:1.10-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make

FROM gobuild-base AS base
WORKDIR /go/src/github.com/scratch/scratch
COPY . .
RUN make static && mv scratch /usr/bin/scratch

FROM scratch
COPY --from=base /usr/bin/scratch /usr/bin/scratch
ENTRYPOINT [ "scratch" ]
CMD [ ]`)
	bomPrefixDockerfile := append(utf8BOM, df...)
	runtimeDep, buildDeps, err := resolveDockerfileDependencies(bytes.NewReader(bomPrefixDockerfile), nil, "")
	if err != nil {
		t.Errorf("Failed to resolve dependencies: %v", err)
	}
	if runtimeDep != expectedRuntime {
		t.Errorf("Unexpected runtime. Got %s, expected %s", runtimeDep, expectedRuntime)
	}
	for _, buildDep := range buildDeps {
		if ok := expectedBuildDeps[buildDep]; !ok {
			t.Errorf("Unexpected build-time dependencies. Got %v which wasn't expected", buildDep)
		}
	}
}

func TestResolveDockerfileDependencies_WithTarget(t *testing.T) {
	expectedRuntime := "node:lts"
	expectedBuildDeps := map[string]bool{
		fmt.Sprintf("scratch"): true,
	}
	df := []byte(`FROM scratch AS base
	RUN ls
	FROM node:lts AS build
	RUN ls
	FROM nginx:stable AS final
	RUN ls`)
	runtimeDep, buildDeps, err := resolveDockerfileDependencies(bytes.NewReader(df), nil, "build")
	if err != nil {
		t.Errorf("Failed to resolve dependencies: %v", err)
	}
	if runtimeDep != expectedRuntime {
		t.Errorf("Unexpected runtime. Got %s, expected %s", runtimeDep, expectedRuntime)
	}
	for _, buildDep := range buildDeps {
		if ok := expectedBuildDeps[buildDep]; !ok {
			t.Errorf("Unexpected build-time dependencies. Got %v which wasn't expected", buildDep)
		}
	}
}

func TestCreateDockerfilePath(t *testing.T) {
	tests := []struct {
		context    string
		workDir    string
		dockerfile string
		expected   string
	}{
		// Remote context
		{"https://github.com/Azure/acr-builder.git", "", "", defaultDockerfile},
		{"https://github.com/Azure/acr-builder.git", "build", "", "build/" + defaultDockerfile},
		{"https://github.com/Azure/acr-builder.git#:foo/bar", "", "foo/bar/Dockerfile", "foo/bar/Dockerfile"},
		{"https://github.com/Azure/acr-builder.git#:foo/bar", "build", "foo/bar/Dockerfile", "build/foo/bar/Dockerfile"},

		// Local context
		{".", ".", "", defaultDockerfile},
		{"src/foo", "src/foo", "", "src/foo/" + defaultDockerfile},
		{"src/foo", "src/foo", "bar/qux/Dockerfile", "bar/qux/Dockerfile"},
	}

	for _, test := range tests {
		if actual := createDockerfilePath(test.context, test.workDir, test.dockerfile); actual != test.expected {
			t.Errorf("expected %s but got %s", test.expected, actual)
		}
	}
}

func TestNewImageReference(t *testing.T) {
	tests := []struct {
		imagePath              string
		expectedImageReference *image.Reference
	}{
		{
			"registry.hub.docker.com/library/node:16",
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "registry.hub.docker.com/library/node:16",
			},
		},
		{
			"library/node:16",
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "library/node:16",
			},
		},
		{
			"node:16",
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "node:16",
			},
		},
		{
			"myregistry.azurecr.io/myorg/myimage:mytag",
			&image.Reference{
				Registry:   "myregistry.azurecr.io",
				Repository: "myorg/myimage",
				Tag:        "mytag",
				Reference:  "myregistry.azurecr.io/myorg/myimage:mytag",
			},
		},
		{
			"myorg/myimage:mytag",
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "myorg/myimage",
				Tag:        "mytag",
				Reference:  "myorg/myimage:mytag",
			},
		},
		{
			"registry.hub.docker.com/myorg/myimage:mytag",
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "myorg/myimage",
				Tag:        "mytag",
				Reference:  "registry.hub.docker.com/myorg/myimage:mytag",
			},
		},
	}

	for _, test := range tests {
		imageReference, err := NewImageReference(test.imagePath)
		if err != nil {
			t.Errorf("Failed to get image reference: %v", err)
		} else {
			if imageReference.Registry != test.expectedImageReference.Registry {
				t.Errorf("Unexpected registry. Got %s, expected %s", imageReference.Registry, test.expectedImageReference.Registry)
			}
			if imageReference.Repository != test.expectedImageReference.Repository {
				t.Errorf("Unexpected repository. Got %s, expected %s", imageReference.Repository, test.expectedImageReference.Repository)
			}
			if imageReference.Tag != test.expectedImageReference.Tag {
				t.Errorf("Unexpected tag. Got %s, expected %s", imageReference.Tag, test.expectedImageReference.Tag)
			}
			if imageReference.Reference != test.expectedImageReference.Reference {
				t.Errorf("Unexpected reference. Got %s, expected %s", imageReference.Reference, test.expectedImageReference.Reference)
			}
		}
	}
}
