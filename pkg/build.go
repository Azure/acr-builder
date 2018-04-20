package build

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/docker/distribution/reference"
)

// DockerHubRegistry is the docker hub registry
const DockerHubRegistry = "registry.hub.docker.com"

// EnvVar defines an environmental variable
type EnvVar struct {
	Name  string
	Value string
}

var envVarRegex = regexp.MustCompile("^[a-zA-z_][a-zA-z_0-9]*$")

// NewEnvVar creates an environmental variable with error checking
func NewEnvVar(name string, value string) (*EnvVar, error) {
	if !envVarRegex.Match([]byte(name)) {
		return nil, fmt.Errorf("Invalid environmental variable name: %s", name)
	}
	return &EnvVar{Name: name, Value: value}, nil
}

// Request defines a acr-builder build
type Request struct {
	DockerRegistry    string
	DockerCredentials []DockerCredential
	Targets           []SourceTarget
}

// SourceTarget defines a source and a set of BuildTargets to run
type SourceTarget struct {
	Source Source
	Builds []Target
}

// Source defines where the source code is and how to fetch the code
type Source interface {
	Obtain(runner Runner) error
	Return(runner Runner) error
	Export() []EnvVar
}

// Target is the build part of SourceTarget
type Target interface {
	// Build task can't be a generic tasks now because it needs to return ImageDependencies
	// If we use docker events to figure out dependencies, we can make build tasks a generic task
	Build(runner Runner) error
	Push(runner Runner) error
	ScanForDependencies(runner Runner) ([]ImageDependencies, error)
	Export() []EnvVar
}

// ImageReference defines the reference to a docker image
type ImageReference struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest"`
	reference  reference.Reference
}

// GitReference defines the reference to git source code
type GitReference struct {
	GitHeadRev string `json:"git-head-revision"`
}

// NewImageReference parses a path of a image and creates a ImageReference object
func NewImageReference(path string) (*ImageReference, error) {
	ref, err := reference.Parse(path)
	if err != nil {
		return nil, err
	}
	result := &ImageReference{
		reference: ref,
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
				result.Repository = reference.Path(named)
			} else {
				// The domain is the DockerHub user name
				result.Registry = DockerHubRegistry
				result.Repository = strings.Join([]string{reference.Domain(named), reference.Path(named)}, "/")
			}
		}
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		result.Tag = tagged.Tag()
	} else {
		// Append the "latest" Tag if the path doesn't have a tag AND
		// it isn't a manifest. E.g., the path is "golang".
		// If the user specified a digest they shouldn't receive automatic updates.
		if !strings.Contains(path, "@") {
			result.Tag = "latest"
		}
	}
	return result, nil
}

// String method converts the ImageReference to string
func (ref *ImageReference) String() string {
	return ref.reference.String()
}

// ImageDependencies denotes docker image dependencies
type ImageDependencies struct {
	Image     *ImageReference   `json:"image"`
	Runtime   *ImageReference   `json:"runtime-dependency"`
	Buildtime []*ImageReference `json:"buildtime-dependency"`
	Git       *GitReference     `json:"git,omitempty"`
}

// NewImageDependencies creates ImageDependencies with no references registered
func NewImageDependencies(env *BuilderContext, image string, runtime string, buildtimes []string) (*ImageDependencies, error) {

	var dependencies *ImageDependencies
	if len(image) > 0 {
		image = env.Expand(image)
		imageReference, err := NewImageReference(image)
		if err != nil {
			return nil, err
		}
		dependencies = &ImageDependencies{
			Image: imageReference,
		}
	} else {
		// we allow build without pushing image to registry so the image can be empty
		dependencies = &ImageDependencies{
			Image: nil,
		}
	}

	runtimeDep, err := NewImageReference(env.Expand(runtime))
	if err != nil {
		return nil, err
	}
	dependencies.Runtime = runtimeDep

	for _, buildtime := range buildtimes {
		buildtimeDep, err := NewImageReference(env.Expand(buildtime))
		if err != nil {
			return nil, err
		}
		dependencies.Buildtime = append(dependencies.Buildtime, buildtimeDep)
	}

	// gitSource.Obtain will set the env. For other sources, it will be empty.
	gitHeadRev := os.Getenv(constants.ExportsGitHeadRev)

	if len(gitHeadRev) > 0 {
		dependencies.Git = &GitReference{
			GitHeadRev: gitHeadRev,
		}
	}

	return dependencies, nil
}

// DockerCredential denote how to authenticate to a docker registry
type DockerCredential interface {
	Authenticate(runner Runner) error
}
