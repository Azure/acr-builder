// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/pkg/procmanager"

	"github.com/Azure/acr-builder/util"
)

func TestNewDockerVolumeHelper(t *testing.T) {
	tests := []struct {
		name               string
		expectedCreateArgs []string
		expectedDeleteArgs []string
	}{
		{
			"foo",
			[]string{"docker", "volume", "create", "--name", "foo"},
			[]string{"docker", "volume", "rm", "foo"},
		},
	}
	procManager := procmanager.NewProcManager(true)

	for _, test := range tests {
		vol := NewDockerVolumeHelper(test.name, procManager)
		if vol.Name != test.name {
			t.Fatalf("Expected volume name: %s but got %s", test.name, vol.Name)
		}
		if actual := vol.getDockerCreateArgs(); !util.StringSequenceEquals(actual, test.expectedCreateArgs) {
			t.Fatalf("Expected %v as the create args, but got %v", test.expectedCreateArgs, actual)
		}
		if actual := vol.getDockerRmArgs(); !util.StringSequenceEquals(actual, test.expectedDeleteArgs) {
			t.Fatalf("Expected %v as the delete args, but got %v", test.expectedCreateArgs, actual)
		}

		out, err := vol.Create(context.Background())
		if err != nil {
			t.Fatalf("Unexpected err during volume creation: %v", err)
		}
		if out != "" {
			t.Fatalf("Unexpected output from volume creation: %s", out)
		}

		out, err = vol.Delete(context.Background())
		if err != nil {
			t.Fatalf("Unexpected err during volume deletion: %v", err)
		}
		if out != "" {
			t.Fatalf("Unexpected output from volume deletion: %s", out)
		}
	}
}
