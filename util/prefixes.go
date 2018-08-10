package util

import (
	"fmt"
	"strings"
)

// PrefixRegistryToImageName prefixes the specified registry to the image.
func PrefixRegistryToImageName(registry string, img string) string {
	if registry == "" {
		return img
	}

	if !strings.HasPrefix(img, registry) && !strings.HasPrefix(img, "library/") {
		return fmt.Sprintf("%s/%s", registry, img)
	}

	return img
}
