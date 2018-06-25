package models

import "github.com/docker/distribution/reference"

// ImageDependencies denotes docker image dependencies
type ImageDependencies struct {
	Image     *ImageReference   `json:"image"`
	Runtime   *ImageReference   `json:"runtime-dependency"`
	Buildtime []*ImageReference `json:"buildtime-dependency"`
	Git       *GitReference     `json:"git,omitempty"`
}

// ImageReference defines the reference to a docker image
type ImageReference struct {
	Registry   string              `json:"registry"`
	Repository string              `json:"repository"`
	Tag        string              `json:"tag,omitempty"`
	Digest     string              `json:"digest"`
	Reference  reference.Reference `json:"-"`
}

// GitReference defines the reference to git source code
type GitReference struct {
	GitHeadRev string `json:"git-head-revision"`
}

// String method converts the ImageReference to string
func (ref *ImageReference) String() string {
	return ref.Reference.String()
}
