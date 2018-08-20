// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package image

import (
	"fmt"
)

// Dependencies denotes docker image dependencies.
type Dependencies struct {
	Image     *Reference    `json:"image"`
	Runtime   *Reference    `json:"runtime-dependency"`
	Buildtime []*Reference  `json:"buildtime-dependency"`
	Git       *GitReference `json:"git,omitempty"`
}

// Reference defines the reference to a docker image
type Reference struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest"`
	Reference  string `json:"reference"`
}

// ReferencesEquals determines if two image references are equal.
func ReferencesEquals(img1 *Reference, img2 *Reference) bool {
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
func (i *Reference) String() string {
	if i == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Registry: %s\nRepository: %s\nTag: %s\nDigest: %s\nReference: %s\n", i.Registry, i.Repository, i.Tag, i.Digest, i.Reference)
}

// GitReference defines the reference to git source code
type GitReference struct {
	GitHeadRev string `json:"git-head-revision"`
}
