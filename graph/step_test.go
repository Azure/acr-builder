// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package graph

import "testing"

func TestIsBuildStep(t *testing.T) {
	tests := []struct {
		step     *Step
		expected bool
	}{
		{
			&Step{
				Run: "build -t foo .",
			},
			true,
		},
		{
			&Step{
				Run: "builder build -t foo .",
			},
			false,
		},
		{
			&Step{
				Run: "build",
			},
			false,
		},
	}

	for _, test := range tests {
		if actual := test.step.IsBuildStep(); actual != test.expected {
			t.Errorf("Expected step build step to be %v, but got %v", test.expected, actual)
		}
	}
}
