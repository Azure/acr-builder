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
	windowsOS               = "windows"
)

var (
	errMissingID       = errors.New("step is missing an ID")
	errMissingProps    = errors.New("step is missing a cmd, build, or push property")
	errIDContainsSpace = errors.New("step ID cannot contain spaces")
	errInvalidDeps     = errors.New("step cannot contain other IDs in when if the immediate execution token is specified")
	errInvalidStepType = errors.New("step must only contain a single build, cmd, or push property")
	errInvalidRetries  = errors.New("step must specify retries >= 0")
	errInvalidRepeat   = errors.New("step must specify repeat >= 0")
)

// Step is a step in the execution task.
type Step struct {
	ID                  string   `yaml:"id"`
	Cmd                 string   `yaml:"cmd"`
	Build               string   `yaml:"build"`
	WorkingDirectory    string   `yaml:"workingDirectory"`
	EntryPoint          string   `yaml:"entryPoint"`
	User                string   `yaml:"user"`
	Network             string   `yaml:"network"`
	Isolation           string   `yaml:"isolation"`
	Push                []string `yaml:"push"`
	Envs                []string `yaml:"env"`
	Expose              []string `yaml:"expose"`
	Ports               []string `yaml:"ports"`
	When                []string `yaml:"when"`
	ExitedWith          []int    `yaml:"exitedWith"`
	ExitedWithout       []int    `yaml:"exitedWithout"`
	Timeout             int      `yaml:"timeout"`
	StartDelay          int      `yaml:"startDelay"`
	RetryDelayInSeconds int      `yaml:"retryDelay"`
	// Retries specifies how many times a Step will be retried if it fails after its initial execution.
	Retries int `yaml:"retries"`
	// Repeat specifies how many times a Step will be repeated after its initial execution.
	Repeat                          int  `yaml:"repeat"`
	Keep                            bool `yaml:"keep"`
	Detach                          bool `yaml:"detach"`
	Privileged                      bool `yaml:"privileged"`
	IgnoreErrors                    bool `yaml:"ignoreErrors"`
	DisableWorkingDirectoryOverride bool `yaml:"disableWorkingDirectoryOverride"`
	Pull                            bool `yaml:"pull"`

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
	if s == nil {
		return nil
	}
	if s.ID == "" {
		return errMissingID
	}
	if s.Retries < 0 {
		return errInvalidRetries
	}
	if s.Repeat < 0 {
		return errInvalidRepeat
	}
	if (s.IsCmdStep() && s.IsPushStep()) || (s.IsCmdStep() && s.IsBuildStep()) || (s.IsBuildStep() && s.IsPushStep()) {
		return errInvalidStepType
	}
	if util.ContainsSpace(s.ID) {
		return errIDContainsSpace
	}
	if !s.IsCmdStep() && !s.IsBuildStep() && !s.IsPushStep() {
		return errMissingProps
	}
	for _, dep := range s.When {
		if dep == ImmediateExecutionToken && len(s.When) > 1 {
			return errInvalidDeps
		}
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
	return s.ID == t.ID &&
		s.Keep == t.Keep &&
		s.Detach == t.Detach &&
		s.Cmd == t.Cmd &&
		s.Build == t.Build &&
		util.StringSequenceEquals(s.Push, t.Push) &&
		s.WorkingDirectory == t.WorkingDirectory &&
		s.EntryPoint == t.EntryPoint &&
		util.StringSequenceEquals(s.Ports, t.Ports) &&
		util.StringSequenceEquals(s.Expose, t.Expose) &&
		util.StringSequenceEquals(s.Envs, t.Envs) &&
		s.Timeout == t.Timeout &&
		util.StringSequenceEquals(s.When, t.When) &&
		util.IntSequenceEquals(s.ExitedWith, t.ExitedWith) &&
		util.IntSequenceEquals(s.ExitedWithout, t.ExitedWithout) &&
		s.StartDelay == t.StartDelay &&
		s.StartTime == t.StartTime &&
		s.EndTime == t.EndTime &&
		s.StepStatus == t.StepStatus &&
		s.Privileged == t.Privileged &&
		s.User == t.User &&
		s.Network == t.Network &&
		s.Isolation == t.Isolation &&
		s.IgnoreErrors == t.IgnoreErrors &&
		s.Retries == t.Retries &&
		s.RetryDelayInSeconds == t.RetryDelayInSeconds &&
		s.DisableWorkingDirectoryOverride == t.DisableWorkingDirectoryOverride &&
		s.Pull == t.Pull &&
		s.Repeat == t.Repeat
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
	if s.IsBuildStep() && runtime.GOOS == windowsOS && !strings.Contains(s.Build, "--isolation") {
		s.Build = fmt.Sprintf("--isolation hyperv %s", s.Build)
	}
}
