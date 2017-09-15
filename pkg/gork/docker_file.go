package gork

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func ResolveDockerfileDependencies(path string) (string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("Error opening dockerfile %s, error: %s", path, err)
	}
	scanner := bufio.NewScanner(file)

	aliases := map[string]string{} // dependency graph
	terminals := map[string]bool{} // terminals are image dependencies that must be pulled during build
	var image string               // cursor for walking graphs
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) > 0 && strings.ToLower(tokens[0]) == "from" {
			if len(tokens) < 2 {
				return "", nil, fmt.Errorf("Unable to understand line %s", line)
			}

			image = tokens[1]
			_, found := aliases[image]
			if !found {
				terminals[image] = true
			}

			if len(tokens) > 2 {
				if len(tokens) < 4 || strings.ToLower(tokens[2]) != "as" {
					return "", nil, fmt.Errorf("Unable to understand line %s", line)
				}
				alias := tokens[3]
				aliases[alias] = image
				// Just ignore the rest of the tokens...
				if len(tokens) > 4 {
					logrus.Infof("Ignoring chunks from FROM clause: %v", tokens[4:])
				}
			}
		}
	}

	// Backtrack to find root dependency of last image reference and we will find the runtime dependency
	// Note that since there is not such thing as forward declaration in multistage syntax we should
	// really never find a cycle in the dependency graph. However, we can check for it for completeness
	for {
		parent, hasParent := aliases[image]
		if !hasParent {
			break
		} else if parent == "" {
			// This really should never happen
			return "", nil, fmt.Errorf("Circular dependencies found while looking for runtime dependency on dockerfile %s", path)
		} else {
			aliases[image] = ""
			image = parent
		}
	}

	buildtimeDependencies := make([]string, 0, len(terminals)-1)
	for terminal := range terminals {
		if terminal != image {
			buildtimeDependencies = append(buildtimeDependencies, terminal)
		}
	}
	return image, buildtimeDependencies, nil
}
