// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import "fmt"

const (
	// The default step timeout is 10 minutes.
	defaultStepTimeoutInSeconds = 60 * 10

	// The minimum step timeout is 10 seconds.
	minStepTimeoutInSeconds = 10

	// The maximum step timeout is 2 hours.
	maxStepTimeoutInSeconds = 60 * 60 * 2

	// The default total timeout is 1 hour.
	defaultTotalTimeoutInSeconds = 60 * 60 * 1

	// The minimum total timeout is 10 seconds.
	minTotalTimeoutInSeconds = 10

	// The maximum total timeout is 6 hours.
	maxTotalTimeoutInSeconds = 60 * 60 * 6
)

// Pipeline represents a build pipeline.
type Pipeline struct {
	Steps        []*Step   `toml:"step"`
	StepTimeout  int       `toml:"stepTimeout,omitempty"`
	TotalTimeout int       `toml:"totalTimeout,omitempty"`
	Push         []string  `toml:"push,omitempty"`
	Secrets      []*Secret `toml:"secrets,omitempty"`
	WorkDir      string    `toml:"workDir,omitempty"`
}

// NewPipeline returns a default Pipeline object.
func NewPipeline(steps []*Step, push []string, secrets []*Secret) *Pipeline {
	p := &Pipeline{
		Steps:        steps,
		StepTimeout:  defaultStepTimeoutInSeconds,
		TotalTimeout: defaultTotalTimeoutInSeconds,
		Push:         push,
		Secrets:      secrets,
	}
	p.initialize()
	return p
}

// initialize normalizes the pipeline's values.
func (p *Pipeline) initialize() {
	if p.StepTimeout <= 0 {
		p.StepTimeout = defaultStepTimeoutInSeconds
	}

	if p.TotalTimeout <= 0 {
		p.TotalTimeout = defaultTotalTimeoutInSeconds
	}

	// Force total timeout to be greater than the individual step timeout.
	if p.TotalTimeout < p.StepTimeout {
		p.TotalTimeout = p.StepTimeout
	}

	if p.StepTimeout < minStepTimeoutInSeconds {
		p.StepTimeout = minStepTimeoutInSeconds
	} else if p.StepTimeout > maxStepTimeoutInSeconds {
		p.StepTimeout = maxStepTimeoutInSeconds
	}

	if p.TotalTimeout < minTotalTimeoutInSeconds {
		p.TotalTimeout = minTotalTimeoutInSeconds
	} else if p.TotalTimeout > maxTotalTimeoutInSeconds {
		p.TotalTimeout = maxTotalTimeoutInSeconds
	}

	for i, s := range p.Steps {
		// If individual steps don't have step timeouts specified,
		// stamp the global timeout on them.
		if s.Timeout == 0 {
			s.Timeout = p.StepTimeout
		}

		if s.ID == "" {
			s.ID = fmt.Sprintf("rally_step_%d", i)
		}

		// Override the step's working directory to be the parent's working directory.
		if s.WorkDir == "" && p.WorkDir != "" {
			s.WorkDir = p.WorkDir
		}

		// Initialize a completion channel for each step.
		if s.CompletedChan == nil {
			s.CompletedChan = make(chan bool)
		}

		// Mark the step as skipped initially
		s.StepStatus = Skipped
	}
}
