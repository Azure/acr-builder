package builder

import (
	"fmt"
	"strings"
)

func prefixRegistryToImageName(registry string, img string) string {
	if registry == "" {
		return img
	}

	if !strings.HasPrefix(img, registry) {
		return fmt.Sprintf("%s/%s", registry, img)
	}

	return img
}
