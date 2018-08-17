package grok

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// ResolveDockerfileDependencies given a docker file, resolve its dependencies
func ResolveDockerfileDependencies(buildArgs []string, path string) (string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("Error opening dockerfile %s, error: %s", path, err)
	}
	scanner := bufio.NewScanner(file)
	context, err := parseBuildArgs(buildArgs)
	if err != nil {
		return "", nil, err
	}
	originLookup := map[string]string{} // given an alias, look up its origin
	allOrigins := map[string]bool{}     // set of all origins
	var origin string
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) > 0 {
			switch strings.ToUpper(tokens[0]) {
			case "FROM":
				if len(tokens) < 2 {
					return "", nil, fmt.Errorf("Unable to understand line %s", line)
				}
				var image = os.Expand(tokens[1], func(key string) string {
					return context[key]
				})
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
					// NOTE: alias cannot contain variables it seems. So we don't call context.Expand on it
					alias := tokens[3]
					originLookup[alias] = origin
					// Just ignore the rest of the tokens...
					if len(tokens) > 4 {
						logrus.Debugf("Ignoring chunks from FROM clause: %v", tokens[4:])
					}
				}
			case "ARG":
				if len(tokens) < 2 {
					return "", nil, fmt.Errorf("Dockerfile syntax requires ARG directive to have exactly 1 argument. LINE: %s", line)
				}
				if strings.Contains(tokens[1], "=") {
					varName, varValue, err := ParseAssignment(tokens[1])
					if err != nil {
						return "", nil, fmt.Errorf("Unable to parse assignment %s, error: %s", tokens[1], err)
					}
					// This line matches docker's behavior here
					// 1. If build arg is passed in, the value will not override
					// 2. It is actually allowed for same ARG to be specified more than once in a Dockerfile
					//    However the subsequent value would be ignored instead of overriding the previous
					if _, found := context[varName]; !found {
						context[varName] = varValue
					}
				}
			}
		}
	}

	if len(allOrigins) == 0 {
		return "", nil, fmt.Errorf("Unexpected dockerfile format")
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

func parseBuildArgs(args []string) (map[string]string, error) {
	result := map[string]string{}
	for _, assignment := range args {
		name, value, err := ParseAssignment(assignment)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}

// ParseAssignment is the helper to help parse an assignment statement, I.E. var=value. No space allowed
func ParseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	val := removeSurroundingQuotes(values[1])
	return values[0], val, nil
}

// removeSurroundingQuotes trims double quotes, then single quotes.
func removeSurroundingQuotes(s string) string {
	s = strings.Trim(s, `"`)
	return strings.Trim(s, `'`)
}
