package grok

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// ResolveDockerfileDependencies given a docker file, resolve its dependencies
func ResolveDockerfileDependencies(path string) (string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("Error opening dockerfile %s, error: %s", path, err)
	}
	scanner := bufio.NewScanner(file)

	originLookup := map[string]string{} // given an alias, look up its origin
	allOrigins := map[string]bool{}     // set of all origins
	var origin string
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) > 0 && strings.ToLower(tokens[0]) == "from" {
			if len(tokens) < 2 {
				return "", nil, fmt.Errorf("Unable to understand line %s", line)
			}

			var image = tokens[1]
			var found bool
			origin, found = originLookup[image]
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

	// note that origin variable now points to the runtime origin

	buildtimeDependencies := make([]string, 0, len(allOrigins)-1)
	for terminal := range allOrigins {
		if terminal != origin {
			buildtimeDependencies = append(buildtimeDependencies, terminal)
		}
	}
	// assert len(buildtimeDependencies) == len(allOrigins)-1
	return origin, buildtimeDependencies, nil
}
