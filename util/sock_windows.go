// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

// GetDockerSock returns a volume mapping to the Docker named pipe.
func GetDockerSock() string {
	return "\\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine"
}
