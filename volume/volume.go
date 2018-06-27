// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"context"
	"sync"

	"github.com/Azure/acr-builder/cmder"
)

const (
	// VolumePrefix is a prefix for all Rally volumes.
	VolumePrefix = "rally_vol_"
)

// Volume describes a Docker volume.
type Volume struct {
	Name  string
	cmder *cmder.Cmder
	mu    sync.Mutex
}

// NewVolume creates a new Volume.
func NewVolume(name string, c *cmder.Cmder) *Volume {
	return &Volume{
		Name:  name,
		cmder: c,
		mu:    sync.Mutex{},
	}
}

// Create creates a Docker volume representing the Volume.
func (v *Volume) Create(ctx context.Context) error {
	cmd := []string{"docker", "volume", "create", "--name", v.Name}
	return v.cmder.Run(ctx, cmd, nil, nil, nil, "")
}

// Delete deletes the associated Docker volume.
func (v *Volume) Delete(ctx context.Context) error {
	cmd := []string{"docker", "volume", "rm", v.Name}
	return v.cmder.Run(ctx, cmd, nil, nil, nil, "")
}
