package cli

import "github.com/Azure/acr-builder/builder"

// getNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func getNormalizedDockerImageNames(dockerImages []string) []string {
	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, d := range dockerImages {
		d := builder.NormalizeImageTag(d)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}
