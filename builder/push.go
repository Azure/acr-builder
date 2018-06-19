package builder

import (
	"context"
	"os"
)

// Push attempts to push the specified image names
func (b *Builder) Push(ctx context.Context, imageNames []string) error {
	registry := b.buildOptions.RegistryName

	// TODO: parallel push
	// TODO: retry push after general failures
	for _, img := range imageNames {
		img = prefixRegistryToImageName(registry, img)

		args := []string{"docker", "push", img}
		if err := b.cmder.Run(ctx, args, nil, os.Stdout, os.Stderr, ""); err != nil {
			return err
		}
	}

	return nil
}
