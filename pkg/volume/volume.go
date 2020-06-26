// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"regexp"

	"github.com/pkg/errors"
)

// Volume describes a Docker bind mounted volume.
type Volume struct {
	Name   string              `yaml:"name"`
	Values []map[string]string `yaml:"secret"`
}

//Validate checks whether Volume is well formed
func (v *Volume) Validate() error {
	if v == nil {
		return nil
	}
	if v.Name == "" || len(v.Values) <= 0 {
		return errors.New("volume name or secret is empty")
	}
	var IsCorrectVolumeName = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString
	if !IsCorrectVolumeName(v.Name) {
		return errors.New("volume name is not well formed. Only use alphanumeric and - _")
	}
	for _, values := range v.Values {
		if values != nil {
			if len(values) > 1 {
				return errors.New("each new <secret_name:value> mapping must start as a new element of list")
			}
			for k := range values {
				if k == "" {
					return errors.New("secret name provided for value is empty")
				}
			}
		}
	}
	return nil
}
