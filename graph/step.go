package graph

import (
	"errors"
	"fmt"
	"time"

	"github.com/Azure/acr-builder/util"
)

const (
	// ImmediateExecutionToken defines the whenAll dependency to indicate a step should execute immediately.
	ImmediateExecutionToken = "-"
)

var (
	errMissingID  = errors.New("Step is missing an ID")
	errMissingRun = errors.New("Step is missing a `run` section")
)

// Step is a step in the execution pipeline.
type Step struct {
	ID            string   `toml:"id"`
	Run           string   `toml:"run"`
	ContextDir    string   `toml:"contextDir"`
	WorkDir       string   `toml:"workDir"`
	EntryPoint    string   `toml:"entryPoint"`
	Envs          []string `toml:"envs"`
	SecretEnvs    []string `toml:"secretEnvs"`
	Timeout       int      `toml:"timeout"`
	WhenAll       []string `toml:"whenAll"`
	ExitedWith    []int    `toml:"exitedWith"`
	ExitedWithout []int    `toml:"exitedWithout"`

	StartTime  time.Time
	EndTime    time.Time
	StepStatus StepStatus

	// CompletedChan can be used to signal to readers
	// that the step has been processed.
	CompletedChan chan bool
}

// Validate validates the step and returns an error if the Step has problems.
func (s *Step) Validate() error {
	if s.ID == "" {
		return errMissingID
	}

	if s.Run == "" {
		return errMissingRun
	}

	for _, dep := range s.WhenAll {
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
		s.Run != t.Run ||
		s.WorkDir != t.WorkDir ||
		s.ContextDir != t.ContextDir ||
		s.EntryPoint != t.EntryPoint ||
		!util.StringSequenceEquals(s.Envs, t.Envs) ||
		!util.StringSequenceEquals(s.SecretEnvs, t.SecretEnvs) ||
		s.Timeout != t.Timeout ||
		!util.StringSequenceEquals(s.WhenAll, t.WhenAll) ||
		!util.IntSequenceEquals(s.ExitedWith, t.ExitedWith) ||
		!util.IntSequenceEquals(s.ExitedWithout, t.ExitedWithout) ||
		s.StartTime != t.StartTime ||
		s.EndTime != t.EndTime ||
		s.StepStatus != t.StepStatus {
		return false
	}

	return true
}

// ShouldExecuteImmediately returns true if the Step should be executed immediately.
func (s *Step) ShouldExecuteImmediately() bool {
	if len(s.WhenAll) == 1 && s.WhenAll[0] == ImmediateExecutionToken {
		return true
	}

	return false
}

// HasNoWhenAll returns true if the Step has no whenAll clause, false otherwise.
func (s *Step) HasNoWhenAll() bool {
	return len(s.WhenAll) == 0
}
