package build

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/docker/distribution/reference"
)

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
	Remark(runner Runner, dependencies *ImageDependencies)
	Return(runner Runner) error
	Export() []EnvVar
}

// Target is the build part of SourceTarget
type Target interface {
	// Build task can't be a generic tasks now because it needs to return ImageDependencies
	// If we use docker events to figure out dependencies, we can make build tasks a generic task
	Ensure(runner Runner) error
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
				result.Registry = constants.DockerHubRegistry
				result.Repository = strings.Join([]string{"library", reference.Path(named)}, "/")
			} else {
				// The domain is the DockerHub user name
				result.Registry = constants.DockerHubRegistry
				result.Repository = strings.Join([]string{reference.Domain(named), reference.Path(named)}, "/")
			}
		}
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		result.Tag = tagged.Tag()
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
		imageReference, err := NewImageReference(NormalizeImageTag(image))
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

	runtimeDep, err := NewImageReference(NormalizeImageTag(env.Expand(runtime)))
	if err != nil {
		return nil, err
	}
	dependencies.Runtime = runtimeDep

	dict := map[string]bool{}
	for _, buildtime := range buildtimes {
		bt := NormalizeImageTag(env.Expand(buildtime))

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

// DockerCredential denote how to authenticate to a docker registry
type DockerCredential interface {
	Authenticate(runner Runner) error
}

// NormalizeImageTag adds "latest" to the image if the specified image
// has no tag and it's not referenced by digest.
func NormalizeImageTag(img string) string {
	if !strings.Contains(img, "@") && !strings.Contains(img, ":") {
		return fmt.Sprintf("%s:latest", img)
	}
	return img
}
