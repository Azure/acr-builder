// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"bytes"
	"context"

	"github.com/Azure/acr-builder/pkg/procmanager"
)

const (
	// DockerVolumeHelperPrefix is a prefix for volumes.
	DockerVolumeHelperPrefix = "acb_vol_"
)

// DockerVolumeHelper describes a Docker volume.
type DockerVolumeHelper struct {
	Name        string
	procManager *procmanager.ProcManager
}

// NewDockerVolumeHelper creates a new DockerVolumeHelper.
func NewDockerVolumeHelper(name string, pm *procmanager.ProcManager) *DockerVolumeHelper {
	return &DockerVolumeHelper{
		Name:        name,
		procManager: pm,
	}
}

// Create creates a Docker volume representing the DockerVolumeHelper.
func (v *DockerVolumeHelper) Create(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	err := v.procManager.Run(ctx, v.getDockerCreateArgs(), nil, &buf, &buf, "")
	return buf.String(), err
}

// Delete deletes the associated Docker volume.
func (v *DockerVolumeHelper) Delete(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	err := v.procManager.Run(ctx, v.getDockerRmArgs(), nil, &buf, &buf, "")
	return buf.String(), err
}

func (v *DockerVolumeHelper) getDockerCreateArgs() []string {
	return []string{"docker", "volume", "create", "--name", v.Name}
}

func (v *DockerVolumeHelper) getDockerRmArgs() []string {
	return []string{"docker", "volume", "rm", v.Name}
}
