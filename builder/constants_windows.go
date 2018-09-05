// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

const (
	// homeWorkDir is the working directory to start at in $HOME
	homeWorkDir = "c:\\acb\\home"

	// containerWorkspaceDir is the default working directory for a container.
	containerWorkspaceDir = "c:\\workspace"

	configImageName = "microsoft/windowsservercore-insider@sha256:4517436b65b14b1f497ce4a80668c4ee5161a36342b9574fb33a2e75158c6608"
)

var homeEnv = "USERPROFILE=" + homeWorkDir
