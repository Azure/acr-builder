// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package graph

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		step        *Step
		shouldError bool
	}{
		{
			nil,
			false,
		},
		{
			&Step{},
			true,
		},
		{
			&Step{
				ID: "a",
			},
			true,
		},
		{
			// ID cannot contain spaces.
			&Step{
				ID: "foo bar",
			},
			true,
		},
		{
			&Step{
				ID:   "a",
				Cmd:  "b",
				When: []string{"a"},
			},
			true,
		},
		{
			&Step{
				ID:   "a",
				Cmd:  "b",
				When: []string{"-", "c"},
			},
			true,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
			},
			false,
		},
		{
			&Step{
				ID:   "a",
				Push: []string{"b"},
			},
			false,
		},
		{
			&Step{
				ID:    "a",
				Build: "b",
			},
			false,
		},
		{
			&Step{
				ID:   "a",
				Cmd:  "b",
				Push: []string{"a"},
			},
			true,
		},
		{
			&Step{
				ID:    "a",
				Cmd:   "b",
				Build: "f",
			},
			true,
		},
		{
			&Step{
				ID:    "apple",
				Build: "banana",
				Push:  []string{"d"},
			},
			true,
		},
		{
			&Step{
				ID:     "repeat",
				Build:  "b",
				Repeat: -1,
			},
			true,
		},
		{
			&Step{
				ID:      "retries",
				Build:   "b",
				Retries: -1,
			},
			true,
		},
	}

	for _, test := range tests {
		err := test.step.Validate()
		if test.shouldError && err == nil {
			t.Fatalf("Expected step: %v to error but it didn't", test.step)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("step: %v shouldn't have errored, but it did; err: %v", test.step, err)
		}
	}
}

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

func TestEquals(t *testing.T) {
	tests := []struct {
		s        *Step
		t        *Step
		expected bool
	}{
		{
			nil,
			nil,
			true,
		},
		{
			&Step{
				Cmd: "",
			},
			nil,
			false,
		},
		{
			nil,
			&Step{
				Cmd: "",
			},
			false,
		},
		{
			&Step{
				ID:                              "a",
				Cmd:                             "b",
				Build:                           "c",
				Push:                            []string{"d"},
				WorkingDirectory:                "e",
				EntryPoint:                      "f",
				Envs:                            []string{"g"},
				Expose:                          []string{"j", "k"},
				Ports:                           []string{"l"},
				When:                            []string{"m"},
				ExitedWith:                      []int{0, 1},
				ExitedWithout:                   []int{2},
				Timeout:                         300,
				Keep:                            true,
				Detach:                          false,
				StartDelay:                      1,
				Privileged:                      false,
				User:                            "a",
				Network:                         "b",
				Isolation:                       "c",
				IgnoreErrors:                    false,
				Retries:                         5,
				RetryDelayInSeconds:             3,
				DisableWorkingDirectoryOverride: true,
				Pull:                            true,
				Repeat:                          45,
			},
			&Step{
				ID:                              "a",
				Cmd:                             "b",
				Build:                           "c",
				Push:                            []string{"d"},
				WorkingDirectory:                "e",
				EntryPoint:                      "f",
				Envs:                            []string{"g"},
				Expose:                          []string{"j", "k"},
				Ports:                           []string{"l"},
				When:                            []string{"m"},
				ExitedWith:                      []int{0, 1},
				ExitedWithout:                   []int{2},
				Timeout:                         300,
				Keep:                            true,
				Detach:                          false,
				StartDelay:                      1,
				Privileged:                      false,
				User:                            "a",
				Network:                         "b",
				Isolation:                       "c",
				IgnoreErrors:                    false,
				Retries:                         5,
				RetryDelayInSeconds:             3,
				DisableWorkingDirectoryOverride: true,
				Pull:                            true,
				Repeat:                          45,
			},
			true,
		},
	}

	for _, test := range tests {
		if actual := test.s.Equals(test.t); actual != test.expected {
			t.Errorf("Expected %v and %v to be equal to %v but got %v", test.s, test.t, test.expected, actual)
		}
	}
}

func TestShouldExecuteImmediately(t *testing.T) {
	tests := []struct {
		s        *Step
		expected bool
	}{
		{
			nil,
			false,
		},
		{
			&Step{
				When: []string{},
			},
			false,
		},
		{
			&Step{
				When: nil,
			},
			false,
		},
		{
			&Step{
				When: []string{"a", "b"},
			},
			false,
		},
		{
			&Step{
				When: []string{"-"},
			},
			true,
		},
	}

	for _, test := range tests {
		if actual := test.s.ShouldExecuteImmediately(); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}

func TestHasNoWhen(t *testing.T) {
	tests := []struct {
		s        *Step
		expected bool
	}{
		{
			nil,
			true,
		},
		{
			&Step{
				When: []string{},
			},
			true,
		},
		{
			&Step{
				When: nil,
			},
			true,
		},
		{
			&Step{
				When: []string{"a", "b"},
			},
			false,
		},
		{
			&Step{
				When: []string{"-"},
			},
			false,
		},
	}

	for _, test := range tests {
		if actual := test.s.HasNoWhen(); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}
