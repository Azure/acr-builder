// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package vaults

import "context"

// VaultSecret is the interface that provides a secret value stored in a vault.
type VaultSecret interface {
	GetValue(ctx context.Context) (string, error)
}
