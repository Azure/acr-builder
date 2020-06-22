// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package graph

import (
	"strings"
	"testing"

	"github.com/Azure/acr-builder/pkg/volume"
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
		{
			&Step{
				ID:    "a",
				Build: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "c",
						MountPath: "/run/test",
					},
				},
			},
			true,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "c",
						MountPath: "/run/test",
					},
				},
			},
			false,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "c",
						MountPath: "/run/test",
					},
					&volume.Mount{
						Name:      "d",
						MountPath: "/run/test",
					},
				},
			},
			true,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "c",
						MountPath: "/run/test",
					},
					&volume.Mount{
						Name:      "d",
						MountPath: "/run/test2",
					},
				},
			},
			false,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "d",
						MountPath: "/run/test2",
					},
					&volume.Mount{
						Name:      "d",
						MountPath: "/run/test",
					},
				},
			},
			false,
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

func TestValidateMountVolumeNames(t *testing.T) {
	volumes := []*volume.VolumeMount{
		&volume.VolumeMount{
			Name: "vol1",
			Values: []map[string]string{
				{
					"a": "this is a test",
				},
			},
		},
	}
	tests := []struct {
		step        *Step
		shouldError bool
	}{
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "vol1",
						MountPath: "/run/test2",
					},
				},
			},
			false,
		},
		{
			&Step{
				ID:  "a",
				Cmd: "b",
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "vol2",
						MountPath: "/run/test2",
					},
				},
			},
			true,
		},
	}

	for _, test := range tests {
		err := test.step.ValidateMountVolumeNames(volumes)
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

func TestHasMounts(t *testing.T) {
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
				Mounts: []*volume.Mount{},
			},
			false,
		},
		{
			&Step{
				Mounts: []*volume.Mount{
					&volume.Mount{
						Name:      "a",
						MountPath: "/run/test",
					},
				},
			},
			true,
		},
	}

	for _, test := range tests {
		if actual := test.s.HasMounts(); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}
func TestUseBuildCache(t *testing.T) {
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
				Cmd: "",
			},
			false,
		},
		{
			&Step{
				Cache: "blah",
			},
			false,
		},
		{
			&Step{
				Cache: "Enabled",
			},
			false,
		},
		{
			&Step{
				Build: "a",
				Cache: "enabled",
			},
			true,
		},
		{
			&Step{
				Build: "a",
				Cache: "disabled",
			},
			false,
		},
	}

	for _, test := range tests {
		if actual := test.s.UseBuildCacheForBuildStep(); actual != test.expected {
			t.Errorf("Use Build Cache for %v. Expected %v but got %v", test.s, test.expected, actual)
		}
	}
}

func TestGetCmdForBuildCache(t *testing.T) {
	tests := []struct {
		s        *Step
		taskName string
		registry string
		result   string
		ok       bool
	}{
		{
			&Step{
				Tags:  []string{},
				Cache: "disabled",
			},
			"",
			"",
			"",
			false,
		},
		{
			&Step{
				Tags:  []string{"abcd"},
				Cache: "enabled",
			},
			"",
			"sam.azurecr.io",
			"sam.azurecr.io/abcd",
			true,
		},
		{
			&Step{
				ID:    "step0",
				Tags:  []string{"test.com/repo:tag"},
				Cache: "enabled",
			},
			"fooTask",
			"",
			"test.com/repo:cache_fooTask_step0",
			true,
		},
		{
			&Step{
				ID:    "step_acb0",
				Tags:  []string{"test.com/repo:tag"},
				Cache: "enAbled",
			},
			noTaskNamePlaceholder,
			"",
			"test.com/repo:cache_" + noTaskNamePlaceholder + "_step_acb0",
			true,
		},
		{
			&Step{
				ID:    "step_myrandomlylongIDlaskcnlkascnlkanclkansclknaslkcnalkscnlaknsclkalknaslkncalscnlakscnlkascnlkascnlksn",
				Tags:  []string{"test.com/repo:tag"},
				Cache: "enabled",
			},
			"task_myrandomlylongIDlaskcnlkascnlkanclkansclknaslkcnalkscnlaknsclkalknaslkncalscnlakscnlkascnlkascnlksn",
			"",
			"",
			false,
		},
		{
			&Step{
				ID:    "a_b_c",
				Tags:  []string{"test:5000/repo@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
				Cache: "enabled",
			},
			"foo",
			"",
			"test:5000/repo:cache_foo_a_b_c",
			true,
		},
		{
			&Step{
				ID:    "step_acb0",
				Tags:  []string{"foo/foo_bar.com:8080"},
				Cache: "enabled",
			},
			"myTask",
			"",
			"foo/foo_bar.com:cache_myTask_step_acb0",
			true,
		},
		{
			&Step{
				ID:    "1",
				Tags:  []string{"sub-dom1.foo.com/bar/baz/quux"},
				Cache: "enabled",
			},
			noTaskNamePlaceholder,
			"",
			"sub-dom1.foo.com/bar/baz/quux:cache_" + noTaskNamePlaceholder + "_1",
			true,
		},
		{
			&Step{
				ID:    "1",
				Tags:  []string{"sub-dom1.foo.com/bar/baz/quux"},
				Cache: "enabled",
			},
			noTaskNamePlaceholder,
			"",
			"sub-dom1.foo.com/bar/baz/quux:cache_" + noTaskNamePlaceholder + "_1",
			true,
		},
		{
			&Step{
				ID:    "1",
				Tags:  []string{"a", "b", "c", "d"},
				Cache: "enabled",
			},
			noTaskNamePlaceholder,
			"sam.azurecr.io",
			"sam.azurecr.io/a:cache_" + noTaskNamePlaceholder + "_1",
			true,
		},
		{
			&Step{
				ID:    "1",
				Tags:  []string{"a", "sam.azurecr.io/foo", "c", "d"},
				Cache: "enabled",
			},
			noTaskNamePlaceholder,
			"sam.azurecr.io",
			"sam.azurecr.io/foo:cache_" + noTaskNamePlaceholder + "_1",
			true,
		},
	}

	for _, test := range tests {
		actual, err := test.s.GetCmdWithCacheFlags(test.taskName, "sam.azurecr.io")

		if test.ok {
			if err != nil {
				t.Errorf("expected %v to be okay but got an error %v", test.s, err)
			}
		} else {
			if err == nil {
				t.Errorf("expected %v to be errored out but got none", test.s)
			}
		}

		if !strings.Contains(actual, test.result) {
			t.Errorf("step %v could not extract right registry from tags. Expected cache id tag: %v but got %v", test.s, test.result, actual)
		}
	}
}
