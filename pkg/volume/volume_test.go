// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"testing"
)

func TestVolumeValidate(t *testing.T) {
	tests := []struct {
		volumemount *Volume
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&Volume{
				Name: "",
				Source: Source{
					Secret: []map[string]string{
						{
							"a": "this is a test",
						},
					},
				},
			},
			true,
		},
		{
			&Volume{
				Name: "a",
				Source: Source{
					Secret: []map[string]string{
						{
							"b": "this is a test",
						},
					},
				},
			},
			false,
		},
		{
			&Volume{
				Name: "test123-_",
				Source: Source{
					Secret: []map[string]string{
						{
							"b": "this is a test",
						},
					},
				},
			},
			false,
		},
		{
			&Volume{
				Name: "test/.",
				Source: Source{
					Secret: []map[string]string{
						{
							"b": "this is a test",
						},
					},
				},
			},
			true,
		},
	}
	for _, test := range tests {
		err := test.volumemount.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected volume: %v to error but it didn't", test.volumemount)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("volume: %v shouldn't have errored, but it did; err: %v", test.volumemount, err)
		}
	}
}
