// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

// Secret defines a wrapper to translate Azure Key Vault secrets to environment variables.
type Secret struct {
	Akv       string `yaml:"akv,omitempty"`
	SecretEnv string `yaml:"secretEnv,omitempty"`
}
