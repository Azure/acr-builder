// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"regexp"

	"github.com/pkg/errors"
)

// Source represents the source of a volume to mount.
// Only one of its members may be specified.
type Source struct {
	// Secret represents a secret that should populate this volume.
	Secret map[string]string `yaml:"secret,omitempty"`
	// add more sources here ...
}

// Validate checks whether Source is well formed
func (s *Source) Validate() error {
	if s == nil {
		return nil
	}
	if s.Secret == nil {
		return errors.New("currently only support for source type secret")
	}
	if len(s.Secret) <= 0 {
		return errors.New("secret is empty")
	}
	var IsCorrectSecretFileName = regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`).MatchString
	for key := range s.Secret {
		if key == "" {
			return errors.New("secret name provided for value is empty")
		}
		if !IsCorrectSecretFileName(key) {
			return errors.New("file name, " + key + ", is not well formed. Only use alphanumeric and - _ .")
		}
	}
	return nil
}
