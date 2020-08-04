// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"fmt"
	"strings"
)

// PrefixTags prefixes tags in the specified command and returns the new command.
func PrefixTags(cmd string, registry string, allKnownRegistries []string) (joinedTags string, tags []string) {
	fields := strings.Fields(cmd)
	for i := 1; i < len(fields); i++ {
		if fields[i-1] == "-t" || fields[i-1] == "--tag" {
			fields[i] = PrefixRegistryToImageName(registry, fields[i], allKnownRegistries)
			tags = append(tags, fields[i])
		}
	}
	return strings.Join(fields, " "), tags
}

// PrefixRegistryToImageName prefixes the specified registry to the image.
func PrefixRegistryToImageName(registry string, img string, allKnownRegistries []string) string {
	if registry == "" {
		return img
	}

	if !hasKnownRegistryPrefix(img, allKnownRegistries) && !strings.HasPrefix(img, "library/") {
		return fmt.Sprintf("%s/%s", registry, img)
	}

	return img
}

func hasKnownRegistryPrefix(img string, allKnownRegistries []string) bool {
	for _, registry := range allKnownRegistries {
		if registry != "" && strings.HasPrefix(img, registry) {
			return true
		}
	}

	return false
}

// NormalizeImageTag adds "latest" to the image if the specified image
// has no tag and it's not referenced by digest.
func NormalizeImageTag(img string) string {
	if !strings.Contains(img, "@") && !strings.Contains(img, ":") {
		return fmt.Sprintf("%s:latest", img)
	}
	return img
}
