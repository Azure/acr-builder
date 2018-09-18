// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"io/ioutil"
	"runtime"

	"github.com/Azure/acr-builder/scan"
	"github.com/Azure/acr-builder/util"
	yaml "gopkg.in/yaml.v2"

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

	// The default step retry delay is 5 seconds.
	defaultStepRetryDelayInSeconds = 5
)

// Task represents a task execution.
type Task struct {
	Steps            []*Step    `yaml:"steps"`
	StepTimeout      int        `yaml:"stepTimeout,omitempty"`
	TotalTimeout     int        `yaml:"totalTimeout,omitempty"`
	Secrets          []*Secret  `yaml:"secrets,omitempty"`
	Networks         []*Network `yaml:"networks,omitempty"`
	WorkingDirectory string     `yaml:"workingDirectory,omitempty"`
	Version          string     `yaml:"version,omitempty"`
	RegistryName     string
	RegistryUsername string
	RegistryPassword string
	Dag              *Dag
	IsBuildTask      bool // Used to skip the default network creation for build.
}

// UnmarshalTaskFromString unmarshals a Task from a raw string.
func UnmarshalTaskFromString(data, registry, user, pw string) (*Task, error) {
	t := &Task{}
	if err := yaml.Unmarshal([]byte(data), t); err != nil {
		return t, errors.Wrap(err, "failed to deserialize task")
	}
	t.setRegistryInfo(registry, user, pw)
	err := t.initialize()
	return t, err
}

// UnmarshalTaskFromFile unmarshals a Task from a file.
func UnmarshalTaskFromFile(file, registry, user, pw string) (*Task, error) {
	t := &Task{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return t, err
	}
	if err := yaml.Unmarshal([]byte(data), &t); err != nil {
		return t, errors.Wrap(err, "failed to deserialize task")
	}
	t.setRegistryInfo(registry, user, pw)
	err = t.initialize()
	return t, err
}

// NewTask returns a default Task object.
func NewTask(
	steps []*Step,
	secrets []*Secret,
	registry string,
	user string,
	pw string,
	isBuildTask bool) (*Task, error) {
	t := &Task{
		Steps:            steps,
		StepTimeout:      defaultStepTimeoutInSeconds,
		TotalTimeout:     defaultTotalTimeoutInSeconds,
		Secrets:          secrets,
		RegistryName:     registry,
		RegistryUsername: user,
		RegistryPassword: pw,
		IsBuildTask:      isBuildTask,
	}

	err := t.initialize()
	return t, err
}

// initialize normalizes a Task's values.
func (t *Task) initialize() error {
	// Add the default network if none are specified.
	// Only add the default network if we're using tasks.
	addDefaultNetworkToSteps := false
	if !t.IsBuildTask && len(t.Networks) <= 0 {
		defaultNetwork := NewNetwork(DefaultNetworkName, false, "bridge")
		if runtime.GOOS == "windows" {
			defaultNetwork.Driver = "nat"
		}
		t.Networks = append(t.Networks, defaultNetwork)
		addDefaultNetworkToSteps = true
	}

	if t.StepTimeout <= 0 {
		t.StepTimeout = defaultStepTimeoutInSeconds
	}
	if t.TotalTimeout <= 0 {
		t.TotalTimeout = defaultTotalTimeoutInSeconds
	}

	// Force total timeout to be greater than the individual step timeout.
	if t.TotalTimeout < t.StepTimeout {
		t.TotalTimeout = t.StepTimeout
	}

	if t.StepTimeout < minStepTimeoutInSeconds {
		t.StepTimeout = minStepTimeoutInSeconds
	} else if t.StepTimeout > maxStepTimeoutInSeconds {
		t.StepTimeout = maxStepTimeoutInSeconds
	}

	if t.TotalTimeout < minTotalTimeoutInSeconds {
		t.TotalTimeout = minTotalTimeoutInSeconds
	} else if t.TotalTimeout > maxTotalTimeoutInSeconds {
		t.TotalTimeout = maxTotalTimeoutInSeconds
	}

	for i, s := range t.Steps {
		// If individual steps don't have step timeouts specified,
		// stamp the global timeout on them.
		if s.Timeout <= 0 {
			s.Timeout = t.StepTimeout
		}

		if s.Retries > 0 && s.RetryDelayInSeconds <= 0 {
			s.RetryDelayInSeconds = defaultStepRetryDelayInSeconds
		}

		if addDefaultNetworkToSteps && s.Network == "" {
			s.Network = DefaultNetworkName
		}

		if s.ID == "" {
			s.ID = fmt.Sprintf("acb_step_%d", i)
		}

		// Override the step's working directory to be the parent's working directory.
		if s.WorkingDirectory == "" && t.WorkingDirectory != "" {
			s.WorkingDirectory = t.WorkingDirectory
		}

		// Initialize a completion channel for each step.
		if s.CompletedChan == nil {
			s.CompletedChan = make(chan bool)
		}

		// Mark the step as skipped initially
		s.StepStatus = Skipped

		if s.IsBuildStep() {
			s.Tags = util.ParseTags(s.Build)
			s.BuildArgs = util.ParseBuildArgs(s.Build)
		} else if s.IsPushStep() {
			s.Push = getNormalizedDockerImageNames(s.Push, t.RegistryName)
		}
	}

	var err error
	t.Dag, err = NewDagFromTask(t)
	return err
}

// UsingRegistryCreds determines whether or not the Task is using registry creds.
func (t *Task) UsingRegistryCreds() bool {
	return t.RegistryName != "" &&
		t.RegistryPassword != "" &&
		t.RegistryUsername != ""
}

// SetRegistryInfo sets registry information.
func (t *Task) setRegistryInfo(registry, user, pw string) {
	t.RegistryName = registry
	t.RegistryUsername = user
	t.RegistryPassword = pw
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
