// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"fmt"
	"os"
)

const (
	maxPushRetries = 5
)

func (b *Builder) pushWithRetries(ctx context.Context, imageNames []string) error {
	registry := b.buildOptions.RegistryName

	// TODO: parallel push?
	for _, img := range imageNames {
		img = prefixRegistryToImageName(registry, img)
		args := []string{"docker", "push", img}

		retry := 0
		for retry < maxPushRetries {
			fmt.Printf("Pushing image: %s, attempt %d\n", img, retry+1)

			if err := b.cmder.Run(ctx, args, nil, os.Stdout, os.Stderr, ""); err != nil {
				retry++
			} else {
				break
			}
		}

		if retry == maxPushRetries {
			return fmt.Errorf("Failed to push images successfully")
		}
	}

	return nil
}
