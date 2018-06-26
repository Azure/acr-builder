package builder

import (
	"strings"
)

func parseRunArgs(runCmd string, match string) []string {
	fields := strings.Fields(runCmd)
	prevField := ""
	imageNames := []string{}
	for _, field := range fields {
		if prevField == match {
			imageNames = append(imageNames, field)
		}
		prevField = field
	}

	return imageNames
}

// parseDockerBuildCmd parses a docker build command and extracts the
// context and Dockerfile from it.
func parseDockerBuildCmd(cmd string) (dockerfile string, context string) {
	fields := strings.Fields(cmd)
	prev := ""
	dockerfile = "Dockerfile"
	context = "."

	// TODO: support reading from stdin?
	for i := 1; i < len(fields); i++ {
		v := fields[i]

		if prev == "-f" || prev == "--file" {
			dockerfile = v
		} else if !strings.HasPrefix(prev, "-") && !strings.HasPrefix(v, "-") {
			context = v
		}

		prev = v
	}

	return dockerfile, context
}
