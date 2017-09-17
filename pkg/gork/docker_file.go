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

	originLookup := map[string]string{} // given an alias, look up its origin
	allOrigins := map[string]bool{}     // set of all origins
	var image string                    // cursor for walking graphs
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) > 0 && strings.ToLower(tokens[0]) == "from" {
			if len(tokens) < 2 {
				return "", nil, fmt.Errorf("Unable to understand line %s", line)
			}

			image = tokens[1]
			origin, found := originLookup[image]
			if !found {
				allOrigins[image] = true
				origin = image
			}

			if len(tokens) > 2 {
				if len(tokens) < 4 || strings.ToLower(tokens[2]) != "as" {
					return "", nil, fmt.Errorf("Unable to understand line %s", line)
				}
				alias := tokens[3]
				originLookup[alias] = origin
				// Just ignore the rest of the tokens...
				if len(tokens) > 4 {
					logrus.Infof("Ignoring chunks from FROM clause: %v", tokens[4:])
				}
			}
		}
	}

	// Backtrack to find root dependency of last image reference and we will find the runtime dependency
	runtimeOrigin, found := originLookup[image]
	if !found {
		runtimeOrigin = image
		// assert isOrigin[runtimeOrigin] == true
	}

	buildtimeDependencies := make([]string, 0, len(allOrigins)-1)
	for terminal := range allOrigins {
		if terminal != runtimeOrigin {
			buildtimeDependencies = append(buildtimeDependencies, terminal)
		}
	}
	// assert len(buildtimeDependencies) == len(allOrigins)-1
	return runtimeOrigin, buildtimeDependencies, nil
}
