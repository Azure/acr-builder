// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package graph

import (
	"strings"
	"testing"

	"github.com/Azure/acr-builder/util"
)

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

func TestQuoteSplit(t *testing.T) {
	tests := []struct {
		cmd      string
		expected []string
	}{
		{
			cmd:      "a b c d",
			expected: []string{"a", "b", "c", "d"},
		},
		{
			cmd:      `docker run --rm -it bash -c "echo hello-world >> test.txt && ls && cat test.txt"`,
			expected: []string{"docker", "run", "--rm", "-it", "bash", "-c", "echo hello-world >> test.txt && ls && cat test.txt"},
		},
		{
			cmd:      "\"hello world\" > foo.txt",
			expected: []string{"hello world", ">", "foo.txt"},
		},
		{
			cmd:      "bash echo -e \"FROM busybox\nCOPY /hello /\nRUN cat /hello > Dockerfile\"",
			expected: []string{"bash", "echo", "-e", "FROM busybox\nCOPY /hello /\nRUN cat /hello > Dockerfile"},
		},
		{
			cmd:      "docker run --rm -it bash -c \"echo FROM hello-world > dockerfile && ls\"",
			expected: []string{"docker", "run", "--rm", "-it", "bash", "-c", "echo FROM hello-world > dockerfile && ls"},
		},
	}

	for _, test := range tests {
		if actual := toArgv(test.cmd); !util.StringSequenceEquals(test.expected, actual) {
			t.Errorf("Got %v but expected %v after splitting", strings.Join(actual, ","), strings.Join(test.expected, ","))
		}
	}
}
