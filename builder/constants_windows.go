// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

const (
	// homeWorkDir is the working directory to start at in $HOME
	homeWorkDir = "c:\\acb\\home"

	// containerWorkspaceDir is the default working directory for a container.
	containerWorkspaceDir = "c:\\workspace"

	configImageName = "mcr.microsoft.com/windows/nanoserver:ltsc2022"
)

var homeEnv = "USERPROFILE=" + homeWorkDir
