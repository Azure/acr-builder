// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"runtime"
	"testing"

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

func TestCreateFilesForVolume(t *testing.T) {
	pm := procmanager.NewProcManager(false)
	builder := NewBuilder(pm, false, "")
	tests := []struct {
		volumemount *volume.VolumeMount
		shouldError bool
	}{
		{
			&volume.VolumeMount{
				Name: "a",
				Values: []map[string]string{
					{
						"b.txt": "dGhpcyBpcyBhIHRlc3Q=",
					},
				},
			},
			false,
		},
		{
			&volume.VolumeMount{
				Name: "a",
				Values: []map[string]string{
					{
						"b.txt": "this is a test",
					},
				},
			},
			true,
		},
	}
	for _, test := range tests {
		err := builder.createFilesForVolume(context.Background(), test.volumemount)
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
