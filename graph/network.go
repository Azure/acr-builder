// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"bytes"
	"context"
	"runtime"

	"github.com/Azure/acr-builder/pkg/procmanager"
)

const (
	// DefaultNetworkName is the default network name.
	DefaultNetworkName = "acb_default_network"
)

// Network defines a Docker network.
type Network struct {
	Name string `yaml:"name"`
	Ipv6 bool   `yaml:"ipv6"`
}

// NewNetwork creates a new network.
func NewNetwork(name string, ipv6 bool) *Network {
	return &Network{
		Name: name,
		Ipv6: ipv6,
	}
}

// Create creates a new Docker network.
func (n *Network) Create(ctx context.Context, pm *procmanager.ProcManager) (string, error) {
	var buf bytes.Buffer
	cmd := []string{"docker", "network", "create", n.Name}
	if n.Ipv6 {
		cmd = append(cmd, "--ipv6")
	}

	if runtime.GOOS == "windows" {
		cmd = append(cmd, "--driver")
		cmd = append(cmd, "nat")
	}

	err := pm.Run(ctx, cmd, nil, &buf, &buf, "")
	return buf.String(), err
}

// Delete deletes the Docker network.
func (n *Network) Delete(ctx context.Context, pm *procmanager.ProcManager) (string, error) {
	var buf bytes.Buffer
	cmd := []string{"docker", "network", "rm", n.Name}
	err := pm.Run(ctx, cmd, nil, &buf, &buf, "")
	return buf.String(), err
}
