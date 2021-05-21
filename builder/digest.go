package builder

import (
	"context"

	"github.com/Azure/acr-builder/pkg/image"
)

type DigestHelper interface {
	PopulateDigest(ctx context.Context, reference *image.Reference) error
}
