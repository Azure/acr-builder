// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package models

// ImageDependencies denotes docker image dependencies
type ImageDependencies struct {
	Image     *ImageReference   `json:"image"`
	Runtime   *ImageReference   `json:"runtime-dependency"`
	Buildtime []*ImageReference `json:"buildtime-dependency"`
	Git       *GitReference     `json:"git,omitempty"`
}

// ImageReference defines the reference to a docker image
type ImageReference struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest"`
	Reference  string `json:"reference"`
}

// GitReference defines the reference to git source code
type GitReference struct {
	GitHeadRev string `json:"git-head-revision"`
}
