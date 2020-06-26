// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"regexp"

	"github.com/pkg/errors"
)

// Volume describes a Docker bind mounted volume.
type Volume struct {
	Name   string `yaml:"name"`
	Source Source `yaml:",inline"`
}

//Validate checks whether Volume is well formed
func (v *Volume) Validate() error {
	if v == nil {
		return nil
	}
	if v.Name == "" {
		return errors.New("volume name is empty")
	}
	if v.Source.Secret == nil {
		return errors.New("only type Source type Secret supported")
	}
	var IsCorrectVolumeName = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString
	if !IsCorrectVolumeName(v.Name) {
		return errors.New("volume name is not well formed. Only use alphanumeric and - _")
	}
	return nil
}
