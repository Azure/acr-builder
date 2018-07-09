// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

// BuildOptions are configuration options for an Azure Container Registry.
type BuildOptions struct {
	RegistryName     string
	RegistryUsername string
	RegistryPassword string
	Push             bool
}
