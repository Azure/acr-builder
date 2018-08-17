// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"errors"
	"fmt"
	"time"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/Azure/acr-builder/util"
)

const (
	// ImmediateExecutionToken defines the when dependency to indicate a step should execute immediately.
	ImmediateExecutionToken = "-"
)

var (
	errMissingID    = errors.New("Step is missing an ID")
	errMissingProps = errors.New("Step is missing a cmd or build property")
)

// Step is a step in the execution task.
type Step struct {
	ID            string   `yaml:"id"`
	Cmd           string   `yaml:"cmd"`
	Build         string   `yaml:"build"`
	WorkDir       string   `yaml:"workDir"`
	EntryPoint    string   `yaml:"entryPoint"`
	Envs          []string `yaml:"envs"`
	SecretEnvs    []string `yaml:"secretEnvs"`
	Ports         []string `yaml:"ports"`
	When          []string `yaml:"when"`
	ExitedWith    []int    `yaml:"exitedWith"`
	ExitedWithout []int    `yaml:"exitedWithout"`
	Timeout       int      `yaml:"timeout"`
	Keep          bool     `yaml:"keep"`
	Detach        bool     `yaml:"detach"`
	StartDelay    int      `yaml:"startDelay"`
	Privileged    bool     `yaml:"privileged"`
	User          string   `yaml:"user"`
	Network       string   `yaml:"network"`

	StartTime  time.Time
	EndTime    time.Time
	StepStatus StepStatus

	// CompletedChan can be used to signal to readers
	// that the step has been processed.
	CompletedChan chan bool

	ImageDependencies []*models.ImageDependencies
	Tags              []string
	BuildArgs         []string
}

// Validate validates the step and returns an error if the Step has problems.
func (s *Step) Validate() error {
	if s.ID == "" {
		return errMissingID
	}
	if s.Cmd == "" && s.Build == "" {
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
		s.WorkDir != t.WorkDir ||
		s.EntryPoint != t.EntryPoint ||
		!util.StringSequenceEquals(s.Ports, t.Ports) ||
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
		s.Network != t.Network {
		return false
	}

	return true
}

// ShouldExecuteImmediately returns true if the Step should be executed immediately.
func (s *Step) ShouldExecuteImmediately() bool {
	if len(s.When) == 1 && s.When[0] == ImmediateExecutionToken {
		return true
	}

	return false
}

// HasNoWhen returns true if the Step has no when clause, false otherwise.
func (s *Step) HasNoWhen() bool {
	return len(s.When) == 0
}

// IsBuildStep returns true if the Step is a build step, false otherwise.
func (s *Step) IsBuildStep() bool {
	return s.Build != ""
}
