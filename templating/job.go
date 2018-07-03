// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

// Job represents a job to processed by the engine.
type Job struct {
	Config    *Config     `json:"config,omitempty"`
	Templates []*Template `json:"template,omitempty"`
}

// GetConfig returns a Job's config.
func (j *Job) GetConfig() *Config {
	if j == nil {
		return nil
	}
	return j.Config
}

// GetTemplates returns a Job's templates.
func (j *Job) GetTemplates() []*Template {
	if j == nil {
		return nil
	}
	return j.Templates
}

// HasValidConfig determines whether or not a Job has a valid configuration.
func (j *Job) HasValidConfig() bool {
	return j.Config != nil && j.Config.RawValue != ""
}
