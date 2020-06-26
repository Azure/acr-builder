// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package volume

import (
	"testing"
)

func TestSourceValidate(t *testing.T) {
	tests := []struct {
		source      *Source
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&Source{
				Secret: []map[string]string{
					{
						"a": "this is a test",
					},
				},
			},
			false,
		},
		{
			&Source{
				Secret: []map[string]string{},
			},
			true,
		},
		{
			&Source{
				Secret: []map[string]string{
					{
						"": "this is a test",
					},
				},
			},
			true,
		},
		{
			&Source{
				Secret: []map[string]string{
					{
						"a": "",
					},
				},
			},
			false,
		},
	}
	for _, test := range tests {
		err := test.source.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected source: %v to error but it didn't", test.source)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("source: %v shouldn't have errored, but it did; err: %v", test.source, err)
		}
	}
}
