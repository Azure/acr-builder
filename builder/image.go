package builder

import (
	"fmt"
	"strings"

	"github.com/Azure/acr-builder/baseimages/scanner/scan"
)

// GetNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func GetNormalizedDockerImageNames(dockerImages []string) []string {
	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, d := range dockerImages {
		d := scan.NormalizeImageTag(d)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}

func prefixRegistryToImageName(registry string, img string) string {
	if registry == "" {
		return img
	}

	if !strings.HasPrefix(img, registry) {
		return fmt.Sprintf("%s/%s", registry, img)
	}

	return img
}

func prefixStepTags(runCmd string, registry string) string {
	if registry == "" {
		return runCmd
	}

	fields := strings.Fields(runCmd)

	for i := 1; i < len(fields); i++ {
		if fields[i-1] == "-t" {
			fields[i] = prefixRegistryToImageName(registry, fields[i])
		}
	}

	return strings.Join(fields, " ")
}
