// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"github.com/pkg/errors"
)

// Source represents the source of a volume to mount.
// Only one of its members may be specified.
type Source struct {
	// Secret represents a secret that should populate this volume.
	Secret []map[string]string `yaml:"secret,omitempty"`
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
	for _, values := range s.Secret {
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
