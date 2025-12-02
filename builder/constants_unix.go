// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

//go:build linux || darwin

package builder

const (
	// homeWorkDir is the working directory to start at in $HOME.
	homeWorkDir = "/acb/home"

	// containerWorkspaceDir is the default working directory for a container.
	containerWorkspaceDir = "/workspace"

	configImageName = "bash"
)

var homeEnv = "HOME=" + homeWorkDir
