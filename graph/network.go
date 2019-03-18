// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"bytes"
	"context"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/pkg/errors"
)

const (
	// DefaultNetworkName is the default network name.
	DefaultNetworkName = "acb_default_network"
)

var (
	errInvalidName = errors.New("name must be specified")
)

// Network defines a Docker network.
type Network struct {
	Name         string `yaml:"name"`
	Driver       string `yaml:"driver"`
	Ipv6         bool   `yaml:"ipv6"`
	SkipCreation bool   `yaml:"skipCreation"`
	IsDefault    bool   `yaml:"isDefault"`
}

// NewNetwork creates a new network.
func NewNetwork(name string, ipv6 bool, driver string, skipCreation bool, isDefault bool) (*Network, error) {
	if name == "" {
		return nil, errInvalidName
	}
	return &Network{
		Name:         name,
		Ipv6:         ipv6,
		Driver:       driver,
		SkipCreation: skipCreation,
		IsDefault:    isDefault,
	}, nil
}

// Create creates a new Docker network.
func (n *Network) Create(ctx context.Context, pm *procmanager.ProcManager) (string, error) {
	var buf bytes.Buffer
	err := pm.Run(ctx, n.getDockerCreateArgs(), nil, &buf, &buf, "")
	return buf.String(), err
}

// Delete deletes the Docker network.
func (n *Network) Delete(ctx context.Context, pm *procmanager.ProcManager) (string, error) {
	var buf bytes.Buffer
	err := pm.Run(ctx, n.getDockerRmArgs(), nil, &buf, &buf, "")
	return buf.String(), err
}

func (n *Network) getDockerCreateArgs() []string {
	args := []string{"docker", "network", "create", n.Name}
	if n.Ipv6 {
		args = append(args, "--ipv6")
	}
	if n.Driver != "" {
		args = append(args, "--driver", n.Driver)
	}
	return args
}

func (n *Network) getDockerRmArgs() []string {
	return []string{"docker", "network", "rm", n.Name}
}
