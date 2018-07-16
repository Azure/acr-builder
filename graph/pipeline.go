// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"io/ioutil"

	"github.com/Azure/acr-builder/util"
	yaml "gopkg.in/yaml.v2"

	"github.com/Azure/acr-builder/baseimages/scanner/scan"
	"github.com/pkg/errors"
)

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
	Steps            []*Step   `yaml:"steps"`
	StepTimeout      int       `yaml:"stepTimeout,omitempty"`
	TotalTimeout     int       `yaml:"totalTimeout,omitempty"`
	Push             []string  `yaml:"push,omitempty"`
	Secrets          []*Secret `yaml:"secrets,omitempty"`
	WorkDir          string    `yaml:"workDir,omitempty"`
	RegistryName     string
	RegistryUsername string
	RegistryPassword string
	Dag              *Dag
}

// UnmarshalPipelineFromString unmarshals a pipeline from a raw string.
func UnmarshalPipelineFromString(data, registry, user, pw string) (*Pipeline, error) {
	p := &Pipeline{}
	if err := yaml.Unmarshal([]byte(data), p); err != nil {
		return p, errors.Wrap(err, "failed to deserialize pipeline")
	}

	p.setRegistryInfo(registry, user, pw)

	err := p.initialize()
	return p, err
}

// UnmarshalPipelineFromFile unmarshals a pipeline from a file.
func UnmarshalPipelineFromFile(file, registry, user, pw string) (*Pipeline, error) {
	p := &Pipeline{}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return p, err
	}

	if err := yaml.Unmarshal([]byte(data), &p); err != nil {
		return p, err
	}

	p.setRegistryInfo(registry, user, pw)

	err = p.initialize()
	return p, err
}

// NewPipeline returns a default Pipeline object.
func NewPipeline(
	steps []*Step,
	push []string,
	secrets []*Secret,
	registry string,
	user string,
	pw string) (*Pipeline, error) {
	p := &Pipeline{
		Steps:            steps,
		StepTimeout:      defaultStepTimeoutInSeconds,
		TotalTimeout:     defaultTotalTimeoutInSeconds,
		Push:             push,
		Secrets:          secrets,
		RegistryName:     registry,
		RegistryUsername: user,
		RegistryPassword: pw,
	}

	err := p.initialize()
	return p, err
}

// initialize normalizes the pipeline's values.
func (p *Pipeline) initialize() error {
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
			s.ID = fmt.Sprintf("acb_step_%d", i)
		}

		// Override the step's working directory to be the parent's working directory.
		if s.WorkDir == "" && p.WorkDir != "" {
			s.WorkDir = p.WorkDir
		}

		// Initialize a completion channel for each step.
		if s.CompletedChan == nil {
			s.CompletedChan = make(chan bool)
		}

		// Adjust the run command so that the ACR registry is prefixed for all tags
		s.Run = util.PrefixTags(s.Run, p.RegistryName)

		// Mark the step as skipped initially
		s.StepStatus = Skipped

		s.Tags = util.ParseTags(s.Run)
		s.BuildArgs = util.ParseBuildArgs(s.Run)
	}

	p.Push = getNormalizedDockerImageNames(p.Push, p.RegistryName)

	var err error
	p.Dag, err = NewDagFromPipeline(p)

	return err
}

// UsingRegistryCreds determines whether or not the pipeline is using registry creds.
func (p *Pipeline) UsingRegistryCreds() bool {
	return p.RegistryName != "" &&
		p.RegistryPassword != "" &&
		p.RegistryUsername != ""
}

// SetRegistryInfo sets registry information.
func (p *Pipeline) setRegistryInfo(registry, user, pw string) {
	p.RegistryName = registry
	p.RegistryUsername = user
	p.RegistryPassword = pw
}

// getNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func getNormalizedDockerImageNames(dockerImages []string, registry string) []string {
	if len(dockerImages) <= 0 {
		return dockerImages
	}

	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, d := range dockerImages {
		d := scan.NormalizeImageTag(d)
		d = util.PrefixRegistryToImageName(registry, d)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}
