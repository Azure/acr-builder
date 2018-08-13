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
				Build: "-t foo .",
			},
			true,
		},
		{
			&Step{
				Cmd: "builder build -t foo .",
			},
			false,
		},
		{
			&Step{
				Cmd: "build -f Dockerfile -t blah .",
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
