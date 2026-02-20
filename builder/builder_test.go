// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/util"
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

func TestGetRepoDigestDockerHub(t *testing.T) {
	tests := []struct {
		id       int
		json     string
		imgRef   *image.Reference
		expected string
	}{
		{
			1,
			`["registry.hub.docker.com/library/node@sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc"]`,
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "registry.hub.docker.com/library/node:16",
			},
			"sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc",
		},
		{
			2,
			`["node@sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc"]`,
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "node:16",
			},
			"sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc",
		},
		{
			3,
			`["node@sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc"]`,
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "library/node",
				Tag:        "16",
				Reference:  "library/node:16",
			},
			"sha256:466d0a05ecb1e5b9890960592311fa10c2bc6012fc27dbfdcc74abf10fc324fc",
		},
		{
			4,
			`["grafana/grafana@sha256:c2a9d25b77b9a7439e56efffa916e43eda09db4f7b78526082443f9c2ee18dc0"]`,
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "grafana/grafana",
				Tag:        "latest",
				Reference:  "grafana/grafana:latest",
			},
			"sha256:c2a9d25b77b9a7439e56efffa916e43eda09db4f7b78526082443f9c2ee18dc0",
		},
		{
			5,
			`["registry.hub.docker.com/grafana/grafana@sha256:c2a9d25b77b9a7439e56efffa916e43eda09db4f7b78526082443f9c2ee18dc0"]`,
			&image.Reference{
				Registry:   "registry.hub.docker.com",
				Repository: "grafana/grafana",
				Tag:        "latest",
				Reference:  "registry.hub.docker.com/grafana/grafana:latest",
			},
			"sha256:c2a9d25b77b9a7439e56efffa916e43eda09db4f7b78526082443f9c2ee18dc0",
		},
	}
	for _, test := range tests {
		if actual := getRepoDigest(test.json, test.imgRef); actual != test.expected {
			t.Errorf("invalid repo digest, test id: %d; expected %s but got %s", test.id, test.expected, actual)
		}
	}
}

func TestParseImageNameFromArgs(t *testing.T) {
	tests := []struct {
		args     string
		expected string
	}{
		{"bash", "bash"},
		{"", ""},
		{"bash echo hello world", "bash"},
		{"foo bar > qux &", "foo"},
		{"foo    ", "foo"},
	}
	for _, test := range tests {
		if actual := parseImageNameFromArgs(test.args); actual != test.expected {
			t.Errorf("Expected %s but got %s", test.expected, actual)
		}
	}
}

func TestBuildStepDisablesBuildkitWhenNotExplicitlyEnabled(t *testing.T) {
	tests := []struct {
		name            string
		envs            []string
		usesBuildkit    bool
		expectBuildkit0 bool
	}{
		{
			name:            "no envs, buildkit not enabled",
			envs:            nil,
			usesBuildkit:    false,
			expectBuildkit0: true,
		},
		{
			name:            "with envs, buildkit not enabled",
			envs:            []string{"FOO=bar"},
			usesBuildkit:    false,
			expectBuildkit0: true,
		},
		{
			name:            "buildkit explicitly enabled via env",
			envs:            []string{"DOCKER_BUILDKIT=1"},
			usesBuildkit:    true,
			expectBuildkit0: false,
		},
		{
			name:            "buildkit enabled with other envs",
			envs:            []string{"FOO=bar", "DOCKER_BUILDKIT=1"},
			usesBuildkit:    true,
			expectBuildkit0: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := &graph.Step{
				ID:           "build-step",
				Build:        "-f Dockerfile .",
				Envs:         make([]string, len(test.envs)),
				UsesBuildkit: test.usesBuildkit,
			}
			copy(step.Envs, test.envs)

			// Simulate the logic in runStep: inject DOCKER_BUILDKIT=0 when buildkit is not used
			if !step.UsesBuildkit {
				step.Envs = append(step.Envs, "DOCKER_BUILDKIT=0")
			}

			builder := &Builder{}
			args := builder.getDockerRunArgsForStep("volName", "workDir", step, "", "docker build -f Dockerfile .")
			argsStr := strings.Join(args, " ")

			containsBuildkit0 := strings.Contains(argsStr, "DOCKER_BUILDKIT=0")
			if test.expectBuildkit0 && !containsBuildkit0 {
				t.Errorf("expected DOCKER_BUILDKIT=0 in args but was not found: %s", argsStr)
			}
			if !test.expectBuildkit0 && containsBuildkit0 {
				t.Errorf("did not expect DOCKER_BUILDKIT=0 in args but it was found: %s", argsStr)
			}
		})
	}
}

func TestCreateFilesForVolume(t *testing.T) {
	pm := procmanager.NewProcManager(false)
	builder := NewBuilder(pm, false, "")
	tests := []struct {
		volumemount *volume.Volume
		shouldError bool
	}{
		{
			&volume.Volume{
				Name: "a",
				Source: volume.Source{
					Secret: map[string]string{
						"b.txt": "dGhpcyBpcyBhIHRlc3Q=",
					},
				},
			},
			false,
		},
		{
			&volume.Volume{
				Name: "a",
				Source: volume.Source{
					Secret: map[string]string{
						"b.txt": "this is a test",
					},
				},
			},
			true,
		},
	}
	for _, test := range tests {
		err := builder.createSecretFiles(context.Background(), test.volumemount)
		if test.shouldError && err == nil {
			t.Fatalf("Expected file creation of volume mount: %v to error but it didn't", test.volumemount)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("File creation of volume mount: %v shouldn't have errored, but it did; err: %v", test.volumemount, err)
		}
		if !test.shouldError {
			var args []string
			if runtime.GOOS == util.WindowsOS {
				args = []string{"powershell.exe", "-Command", "rm " + test.volumemount.Name + " -r -fo"}
			} else {
				args = []string{"/bin/sh", "-c", "rm -rf " + test.volumemount.Name}
			}
			if err := pm.Run(context.Background(), args, nil, nil, nil, ""); err != nil {
				t.Fatalf("Unexpected err while deleting directory: %v", err)
			}
		}
	}
}
