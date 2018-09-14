// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

const (
	// DockerSocketVolumeMapping returns a volume mapping to the Docker named pipe.
	DockerSocketVolumeMapping = "\\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine"
)
