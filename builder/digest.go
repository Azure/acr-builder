// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"

	"github.com/Azure/acr-builder/pkg/image"
)

type DigestHelper interface {
	PopulateDigest(ctx context.Context, reference *image.Reference) error
}
