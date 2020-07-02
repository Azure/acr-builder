// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"testing"
)

func TestMountValidate(t *testing.T) {
	tests := []struct {
		mount       *Mount
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&Mount{
				Name:      "",
				MountPath: "/run/test",
			},
			true,
		},
		{
			&Mount{
				Name:      "test",
				MountPath: "",
			},
			true,
		},
		{
			&Mount{
				Name:      "",
				MountPath: "",
			},
			true,
		},
		{
			&Mount{
				Name:      "test",
				MountPath: "/run/test",
			},
			false,
		},
	}
	for _, test := range tests {
		err := test.mount.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected mount: %v to error but it didn't", test.mount)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("mount: %v shouldn't have errored, but it did; err: %v", test.mount, err)
		}
	}
}
