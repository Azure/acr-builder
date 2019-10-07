// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

const (
	// DockerSocketVolumeMapping returns a volume mapping to the Docker socket.
	DockerSocketVolumeMapping = "/var/run/docker.sock:/var/run/docker.sock"

	// ContainerdVolumeMapping returns a volume mapping to Containerd.
	ContainerdVolumeMapping = "/run/containerd:/run/containerd"
)
