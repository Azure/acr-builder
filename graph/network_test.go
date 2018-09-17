// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"testing"

	"github.com/Azure/acr-builder/util"
)

func TestNetwork(t *testing.T) {
	tests := []struct {
		name               string
		ipv6               bool
		driver             string
		expectedCreateArgs []string
		expectedDeleteArgs []string
	}{
		{
			"foo",
			true,
			"",
			[]string{"docker", "network", "create", "foo", "--ipv6"},
			[]string{"docker", "network", "rm", "foo"},
		},
		{
			"bar",
			false,
			"nat",
			[]string{"docker", "network", "create", "bar", "--driver", "nat"},
			[]string{"docker", "network", "rm", "bar"},
		},
	}

	for _, test := range tests {
		network := NewNetwork(test.name, test.ipv6, test.driver)
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
	}
}
