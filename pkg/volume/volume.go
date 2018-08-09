// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"bytes"
	"context"

	"github.com/Azure/acr-builder/pkg/procmanager"
)

const (
	// VolumePrefix is a prefix for volumes.
	VolumePrefix = "acb_vol_"
)

// Volume describes a Docker volume.
type Volume struct {
	Name        string
	procManager *procmanager.ProcManager
}

// NewVolume creates a new Volume.
func NewVolume(name string, pm *procmanager.ProcManager) *Volume {
	return &Volume{
		Name:        name,
		procManager: pm,
	}
}

// Create creates a Docker volume representing the Volume.
func (v *Volume) Create(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	cmd := []string{"docker", "volume", "create", "--name", v.Name}
	err := v.procManager.Run(ctx, cmd, nil, &buf, &buf, "")
	return buf.String(), err
}

// Delete deletes the associated Docker volume.
func (v *Volume) Delete(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	cmd := []string{"docker", "volume", "rm", v.Name}
	err := v.procManager.Run(ctx, cmd, nil, &buf, &buf, "")
	return buf.String(), err
}
