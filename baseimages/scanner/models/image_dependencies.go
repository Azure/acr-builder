// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package models

import (
	"fmt"
)

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

// ImageReferencesEquals determines if two image references are equal.
func ImageReferencesEquals(img1 *ImageReference, img2 *ImageReference) bool {
	if img1 == nil && img2 == nil {
		return true
	}
	if img1 == nil || img2 == nil {
		return false
	}

	return img1.Registry == img2.Registry &&
		img1.Repository == img2.Repository &&
		img1.Tag == img2.Tag &&
		img1.Digest == img2.Digest &&
		img1.Reference == img2.Reference
}

// String returns a string representation of an ImageReference.
func (i *ImageReference) String() string {
	if i == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Registry: %s\nRepository: %s\nTag: %s\nDigest: %s\nReference: %s\n", i.Registry, i.Repository, i.Tag, i.Digest, i.Reference)
}

// GitReference defines the reference to git source code
type GitReference struct {
	GitHeadRev string `json:"git-head-revision"`
}
