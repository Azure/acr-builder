// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/util"
)

func TestNetwork(t *testing.T) {
	tests := []struct {
		name               string
		driver             string
		ipv6               bool
		skipCreation       bool
		isDefault          bool
		shouldError        bool
		expectedCreateArgs []string
		expectedDeleteArgs []string
	}{
		{
			"foo",
			"",
			true,
			false,
			false,
			false,
			[]string{"docker", "network", "create", "foo", "--ipv6"},
			[]string{"docker", "network", "rm", "foo"},
		},
		{
			"bar",
			"nat",
			false,
			false,
			false,
			false,
			[]string{"docker", "network", "create", "bar", "--driver", "nat"},
			[]string{"docker", "network", "rm", "bar"},
		},
		{
			"foo",
			"",
			false,
			true,
			true,
			false,
			[]string{"docker", "network", "create", "foo"},
			[]string{"docker", "network", "rm", "foo"},
		},
		{
			"",
			"",
			false,
			true,
			true,
			true, // Test should fail since there's no name.
			[]string{"docker", "network", "create", ""},
			[]string{"docker", "network", "rm", ""},
		},
	}
	procManager := procmanager.NewProcManager(true)

	for _, test := range tests {
		network, err := NewNetwork(test.name, test.ipv6, test.driver, test.skipCreation, test.isDefault)
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test to error but it didn't")
		}
		if err != nil && !test.shouldError {
			t.Fatalf("Test errored when it shouldn't have: %v", err)
		}

		if network.Name != test.name {
			t.Fatalf("Expected network name: %s but got %s", test.name, network.Name)
		}
		if network.Ipv6 != test.ipv6 {
			t.Fatalf("Expected network: %s to have ipv6 of %v, but got %v", test.name, test.ipv6, network.Ipv6)
		}
		if network.Driver != test.driver {
			t.Fatalf("Expected network: %s to have driver of %s, but got %s", test.name, test.driver, network.Driver)
		}
		if actual := network.getDockerCreateArgs(); !util.StringSequenceEquals(actual, test.expectedCreateArgs) {
			t.Fatalf("Expected %v as the create args, but got %v", test.expectedCreateArgs, actual)
		}
		if actual := network.getDockerRmArgs(); !util.StringSequenceEquals(actual, test.expectedDeleteArgs) {
			t.Fatalf("Expected %v as the delete args, but got %v", test.expectedCreateArgs, actual)
		}

		out, err := network.Create(context.Background(), procManager)
		if err != nil {
			t.Fatalf("Unexpected err during network creation: %v", err)
		}
		if out != "" {
			t.Fatalf("Unexpected output from network creation: %s", out)
		}

		out, err = network.Delete(context.Background(), procManager)
		if err != nil {
			t.Fatalf("Unexpected err during network deletion: %v", err)
		}
		if out != "" {
			t.Fatalf("Unexpected output from network deletion: %s", out)
		}
	}
}
