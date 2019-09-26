// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package vault

import "context"

// SecretFetcher is the interface that provides a secret value stored in a vault.
type SecretFetcher interface {
	// FetchSecretValue resolves vault secret values
	FetchSecretValue(ctx context.Context) (string, error)
}
