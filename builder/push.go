package builder

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Push attempts to push the specified image names
func (b *Builder) Push(ctx context.Context, imageNames []string) error {
	registry := b.buildOptions.RegistryName

	// TODO: parallel push
	// TODO: retry push after general failures

	for _, img := range imageNames {
		pushTarget := img
		// If the registry's specified and the image name is already prefixed with
		// the registry's name, don't prefix the registry name again.
		if registry != "" && !strings.HasPrefix(img, registry) {
			pushTarget = fmt.Sprintf("%s/%s", registry, img)
		}

		args := []string{"docker", "push", pushTarget}
		if err := b.cmder.Run(ctx, args, nil, os.Stdout, os.Stderr, ""); err != nil {
			return err
		}
	}

	return nil
}
