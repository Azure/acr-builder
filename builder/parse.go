package builder

import (
	"strings"
)

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

// replacePositionalContext parses the specified command for its positional context
// and replaces it if one's found. Returns the modified command after replacement.
func replacePositionalContext(runCmd string, replacement string) string {
	fields := strings.Fields(runCmd)
	prev := ""

	for i := 1; i < len(fields); i++ {
		if !strings.HasPrefix(prev, "-") && !strings.HasPrefix(fields[i], "-") {
			fields[i] = replacement
			return strings.Join(fields, " ")
		}
		prev = fields[i]
	}

	return runCmd
}
