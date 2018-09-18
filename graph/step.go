// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/util"
)

const (
	// ImmediateExecutionToken defines the when dependency to indicate a step should execute immediately.
	ImmediateExecutionToken = "-"
)

var (
	errMissingID    = errors.New("Step is missing an ID")
	errMissingProps = errors.New("Step is missing a cmd, build, or push property")
)

// Step is a step in the execution task.
type Step struct {
	ID               string   `yaml:"id"`
	Cmd              string   `yaml:"cmd"`
	Build            string   `yaml:"build"`
	Push             []string `yaml:"push"`
	WorkingDirectory string   `yaml:"workingDirectory"`
	EntryPoint       string   `yaml:"entryPoint"`
	Envs             []string `yaml:"env"`
	SecretEnvs       []string `yaml:"secretEnvs"`
	Expose           []string `yaml:"expose"`
	Ports            []string `yaml:"ports"`
	When             []string `yaml:"when"`
	ExitedWith       []int    `yaml:"exitedWith"`
	ExitedWithout    []int    `yaml:"exitedWithout"`
	Timeout          int      `yaml:"timeout"`
	Keep             bool     `yaml:"keep"`
	Detach           bool     `yaml:"detach"`
	StartDelay       int      `yaml:"startDelay"`
	Privileged       bool     `yaml:"privileged"`
	User             string   `yaml:"user"`
	Network          string   `yaml:"network"`
	Isolation        string   `yaml:"isolation"`
	IgnoreErrors     bool     `yaml:"ignoreErrors"`

	StartTime  time.Time
	EndTime    time.Time
	StepStatus StepStatus

	// CompletedChan can be used to signal to readers
	// that the step has been processed.
	CompletedChan chan bool

	ImageDependencies []*image.Dependencies
	Tags              []string
	BuildArgs         []string
}

// Validate validates the step and returns an error if the Step has problems.
func (s *Step) Validate() error {
	if s.ID == "" {
		return errMissingID
	}
	if !s.IsCmdStep() && !s.IsBuildStep() && !s.IsPushStep() {
		return errMissingProps
	}
	for _, dep := range s.When {
		if dep == s.ID {
			return NewSelfReferencedStepError(fmt.Sprintf("Step ID: %v is self-referenced", s.ID))
		}
	}
	return nil
}

// Equals determines whether or not two steps are equal.
func (s *Step) Equals(t *Step) bool {
	if s == nil && t == nil {
		return true
	}
	if s == nil || t == nil {
		return false
	}
	if s.ID != t.ID ||
		s.Keep != t.Keep ||
		s.Detach != t.Detach ||
		s.Cmd != t.Cmd ||
		s.Build != t.Build ||
		!util.StringSequenceEquals(s.Push, t.Push) ||
		s.WorkingDirectory != t.WorkingDirectory ||
		s.EntryPoint != t.EntryPoint ||
		!util.StringSequenceEquals(s.Ports, t.Ports) ||
		!util.StringSequenceEquals(s.Expose, t.Expose) ||
		!util.StringSequenceEquals(s.Envs, t.Envs) ||
		!util.StringSequenceEquals(s.SecretEnvs, t.SecretEnvs) ||
		s.Timeout != t.Timeout ||
		!util.StringSequenceEquals(s.When, t.When) ||
		!util.IntSequenceEquals(s.ExitedWith, t.ExitedWith) ||
		!util.IntSequenceEquals(s.ExitedWithout, t.ExitedWithout) ||
		s.StartDelay != t.StartDelay ||
		s.StartTime != t.StartTime ||
		s.EndTime != t.EndTime ||
		s.StepStatus != t.StepStatus ||
		s.Privileged != t.Privileged ||
		s.User != t.User ||
		s.Network != t.Network ||
		s.Isolation != t.Isolation ||
		s.IgnoreErrors != t.IgnoreErrors {
		return false
	}

	return true
}

// ShouldExecuteImmediately returns true if the Step should be executed immediately.
func (s *Step) ShouldExecuteImmediately() bool {
	if s == nil {
		return false
	}
	if len(s.When) == 1 && s.When[0] == ImmediateExecutionToken {
		return true
	}
	return false
}

// HasNoWhen returns true if the Step has no when clause, false otherwise.
func (s *Step) HasNoWhen() bool {
	if s == nil {
		return true
	}
	return len(s.When) == 0
}

// IsCmdStep returns true if the Step is a command step, false otherwise.
func (s *Step) IsCmdStep() bool {
	if s == nil {
		return false
	}
	return s.Cmd != ""
}

// IsBuildStep returns true if the Step is a build step, false otherwise.
func (s *Step) IsBuildStep() bool {
	if s == nil {
		return false
	}
	return s.Build != ""
}

// IsPushStep returns true if a Step is a push step, false otherwise.
func (s *Step) IsPushStep() bool {
	if s == nil {
		return false
	}
	return len(s.Push) > 0
}

// UpdateBuildStepWithDefaults updates a build step with hyperv isolation on Windows.
func (s *Step) UpdateBuildStepWithDefaults() {
	if s.IsBuildStep() && runtime.GOOS == "windows" && !strings.Contains(s.Build, "--isolation") {
		s.Build = fmt.Sprintf("--isolation hyperv %s", s.Build)
	}
}
