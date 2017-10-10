package domain

import (
	"fmt"
	"regexp"
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

// BuildRequest defines a acr-builder build
type BuildRequest struct {
	DockerRegistry    string
	DockerCredentials []DockerCredential
	Targets           []SourceTarget
}

// SourceTarget defines a source and a set of BuildTargets to run
type SourceTarget struct {
	Source BuildSource
	Builds []BuildTarget
}

// BuildSource defines where the source code is and how to fetch the code
type BuildSource interface {
	Obtain(runner Runner) error
	Return(runner Runner) error
	Export() []EnvVar
}

// BuildTarget is the build part of BuildTarget
type BuildTarget interface {
	// Build task can't be a generic tasks now because it needs to return ImageDependencies
	// If we use docker events to figure out dependencies, we can make build tasks a generic task
	Build(runner Runner) error
	Push(runner Runner) error
	ScanForDependencies(runner Runner) ([]ImageDependencies, error)
	Export() []EnvVar
}

// ImageDependencies denotes docker image dependencies
type ImageDependencies struct {
	Image             string   `json:"image"`
	BuildDependencies []string `json:"build-dependencies"`
	RuntimeDependency string   `json:"runtime-dependency"`
}

// DockerCredential denote how to authenticate to a docker registry
type DockerCredential interface {
	Authenticate(runner Runner) error
}
