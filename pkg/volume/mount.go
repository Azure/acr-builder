// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"github.com/pkg/errors"
)

// Mount describes a volume to be mounted in a particular container
type Mount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
}

// Validate checks whether Mount is well formed
func (m *Mount) Validate() error {
	if m == nil {
		return nil
	}
	if m.Name == "" || m.MountPath == "" {
		return errors.New("mount name or container file path is empty")
	}
	return nil
}
