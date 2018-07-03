// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

// GetDockerSock returns a volume mapping to the Docker socket.
func GetDockerSock() string {
	return "/var/run/docker.sock:/var/run/docker.sock"
}
