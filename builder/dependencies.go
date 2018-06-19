package builder

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/Azure/acr-builder/graph"
	"github.com/docker/distribution/reference"
	"github.com/sirupsen/logrus"
)

// ScanForDependencies scans for base image dependencies.
func (b *Builder) ScanForDependencies(workingDir string, dockerfile string, buildArgs []string, pushTo []string) (deps []*graph.ImageDependencies, err error) {
	path := path.Clean(path.Join(workingDir, dockerfile))
	runtime, buildtime, err := ResolveDockerfileDependencies(path, buildArgs)
	if err != nil {
		return deps, err
	}

	// Even though there's nothing to push to, we always invoke NewImageDependencies
	// TODO: refactor this in the future to take in the full list as opposed to individual
	// images.
	if len(pushTo) <= 0 {
		currDep, err := b.NewImageDependencies("", runtime, buildtime)
		if err != nil {
			return nil, err
		}
		deps = append(deps, currDep)
	}

	for _, imageName := range pushTo {
		currDep, err := b.NewImageDependencies(imageName, runtime, buildtime)
		if err != nil {
			return nil, err
		}
		deps = append(deps, currDep)
	}

	return deps, err
}

// NewImageDependencies creates ImageDependencies with no references registered
func (b *Builder) NewImageDependencies(image string, runtime string, buildtimes []string) (*graph.ImageDependencies, error) {
	var dependencies *graph.ImageDependencies
	if len(image) > 0 {
		imageReference, err := NewImageReference(NormalizeImageTag(image))
		if err != nil {
			return nil, err
		}
		dependencies = &graph.ImageDependencies{
			Image: imageReference,
		}
	} else {
		// we allow build without pushing image to registry so the image can be empty
		dependencies = &graph.ImageDependencies{
			Image: nil,
		}
	}

	runtimeDep, err := NewImageReference(NormalizeImageTag(runtime))
	if err != nil {
		return nil, err
	}
	dependencies.Runtime = runtimeDep

	dict := map[string]bool{}
	for _, buildtime := range buildtimes {
		bt := NormalizeImageTag(buildtime)

		// If the image is prefixed with "library/", remove it for comparisons.
		// "library/" will be added again during image reference generation.
		// This prevents duplicate dependencies when reading "library/golang" and
		// "golang" from the Dockerfile.
		bt = strings.TrimPrefix(bt, "library/")

		// If we've already processed the tag after normalization, skip dependency
		// generation. I.e., they specify "golang" and "golang:latest"
		if dict[bt] {
			continue
		}

		dict[bt] = true

		buildtimeDep, err := NewImageReference(bt)
		if err != nil {
			return nil, err
		}
		dependencies.Buildtime = append(dependencies.Buildtime, buildtimeDep)
	}
	return dependencies, nil
}

// PopulateDigests populates digests on dependencies
func (b *Builder) PopulateDigests(ctx context.Context, dependencies []*graph.ImageDependencies) error {
	for _, entry := range dependencies {
		if err := b.queryDigest(ctx, entry.Image); err != nil {
			return err
		}
		if err := b.queryDigest(ctx, entry.Runtime); err != nil {
			return err
		}
		for _, buildtime := range entry.Buildtime {
			if err := b.queryDigest(ctx, buildtime); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) queryDigest(ctx context.Context, reference *graph.ImageReference) error {
	if reference != nil {
		refString := reference.String()

		// refString will always have the tag specified at this point.
		// For "scratch", we have to compare it against "scratch:latest" even though
		// scratch:latest isn't valid in a FROM clause.
		if refString == NoBaseImageSpecifierLatest {
			return nil
		}

		args := []string{
			"docker",
			"inspect",
			"--format",
			"\"{{json .RepoDigests}}\"",
			refString,
		}

		var buf bytes.Buffer
		err := b.cmder.Run(ctx, args, nil, &buf, os.Stderr, "")
		if err != nil {
			return err
		}

		trimCharPredicate := func(c rune) bool {
			return '\n' == c || '\r' == c || '"' == c || '\t' == c
		}

		reference.Digest = getRepoDigest(strings.TrimFunc(buf.String(), trimCharPredicate), reference)
	}

	return nil
}

func getRepoDigest(jsonContent string, reference *graph.ImageReference) string {
	// Input: ["docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], , docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: ["test.azurecr.io/docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], test.azurecr.io, docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: Invalid
	// Output: <empty>

	prefix := reference.Repository + "@"
	// If the reference has "library/" prefixed, we have to remove it - otherwise
	// we'll fail to query the digest, since image names aren't prefixed with "library/"
	if strings.HasPrefix(prefix, "library/") && reference.Registry == DockerHubRegistry {
		prefix = prefix[8:]
	} else if len(reference.Registry) > 0 && reference.Registry != DockerHubRegistry {
		prefix = reference.Registry + "/" + prefix
	}

	var digestList []string
	if err := json.Unmarshal([]byte(jsonContent), &digestList); err != nil {
		logrus.Warnf("Error deserializing %s to json, error: %s", jsonContent, err)
	}

	for _, digest := range digestList {
		if strings.HasPrefix(digest, prefix) {
			return digest[len(prefix):]
		}
	}

	logrus.Warnf("Unable to find digest for %s in %s", prefix, jsonContent)
	return ""
}

// NormalizeImageTag adds "latest" to the image if the specified image
// has no tag and it's not referenced by digest.
func NormalizeImageTag(img string) string {
	if !strings.Contains(img, "@") && !strings.Contains(img, ":") {
		return fmt.Sprintf("%s:latest", img)
	}
	return img
}

// NewImageReference parses a path of a image and creates a ImageReference object
func NewImageReference(path string) (*graph.ImageReference, error) {
	ref, err := reference.Parse(path)
	if err != nil {
		return nil, err
	}
	result := &graph.ImageReference{
		Reference: ref,
	}

	if named, ok := ref.(reference.Named); ok {
		result.Registry = reference.Domain(named)

		if strings.Contains(result.Registry, ".") {
			// The domain is the registry, eg, registryname.azurecr.io
			result.Repository = reference.Path(named)
		} else {
			// DockerHub
			if result.Registry == "" {
				result.Registry = DockerHubRegistry
				result.Repository = strings.Join([]string{"library", reference.Path(named)}, "/")
			} else {
				// The domain is the DockerHub user name
				result.Registry = DockerHubRegistry
				result.Repository = strings.Join([]string{reference.Domain(named), reference.Path(named)}, "/")
			}
		}
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		result.Tag = tagged.Tag()
	}
	return result, nil
}

// ResolveDockerfileDependencies resolves dependencies given a path to a dockerfile.
func ResolveDockerfileDependencies(path string, buildArgs []string) (string, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("Error opening dockerfile %s, error: %v", path, err)
	}
	defer func() { _ = file.Close() }()

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
					// alias cannot contain variables it seems. So we don't call context.Expand on it
					alias := tokens[3]
					originLookup[alias] = origin
					// Just ignore the rest of the tokens...
					if len(tokens) > 4 {
						fmt.Printf("Ignoring chunks from FROM clause: %v", tokens[4:])
					}
				}
			case "ARG":
				if len(tokens) < 2 {
					return "", nil, fmt.Errorf("Dockerfile syntax requires ARG directive to have exactly 1 argument. LINE: %s", line)
				}
				if strings.Contains(tokens[1], "=") {
					varName, varValue, err := parseAssignment(tokens[1])
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

	return origin, buildtimeDependencies, nil
}

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

func parseBuildArgs(args []string) (map[string]string, error) {
	result := map[string]string{}
	for _, assignment := range args {
		name, value, err := parseAssignment(assignment)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	return result, nil
}

func parseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	return values[0], values[1], nil
}
