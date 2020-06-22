// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"testing"
)

func TestVolumeMountValidate(t *testing.T) {
	tests := []struct {
		volumemount *VolumeMount
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&VolumeMount{
				Name: "",
				Values: []map[string]string{
					{
						"a": "this is a test",
					},
				},
			},
			true,
		},
		{
			&VolumeMount{
				Name:   "test",
				Values: []map[string]string{},
			},
			true,
		},
		{
			&VolumeMount{
				Name:   "",
				Values: []map[string]string{},
			},
			true,
		},
		{
			&VolumeMount{
				Name: "test",
				Values: []map[string]string{
					{
						"a": "this is a test",
					},
				},
			},
			false,
		},
		{
			&VolumeMount{
				Name: "test123-_",
				Values: []map[string]string{
					{
						"a": "this is a test",
					},
				},
			},
			false,
		},
		{
			&VolumeMount{
				Name: "test/.",
				Values: []map[string]string{
					{
						"a": "this is a test",
					},
				},
			},
			true,
		},
		{
			&VolumeMount{
				Name: "test",
				Values: []map[string]string{
					{
						"": "this is a test",
					},
				},
			},
			true,
		},
		{
			&VolumeMount{
				Name: "test",
				Values: []map[string]string{
					{
						"a": "",
					},
				},
			},
			false,
		},
	}
	for _, test := range tests {
		err := test.volumemount.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected volume mount: %v to error but it didn't", test.volumemount)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("volume mount: %v shouldn't have errored, but it did; err: %v", test.volumemount, err)
		}
	}
}
