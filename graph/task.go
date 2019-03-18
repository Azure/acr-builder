// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/Azure/acr-builder/scan"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	// The default step timeout is 10 minutes.
	defaultStepTimeoutInSeconds = 60 * 10

	// The default total timeout is 1 hour.
	defaultTotalTimeoutInSeconds = 60 * 60 * 1

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
	Credentials      []*Credential
	Dag              *Dag
	IsBuildTask      bool     // Used to skip the default network creation for build.
	Envs             []string `yaml:"envs,omitempty"`
}

// UnmarshalTaskFromString unmarshals a Task from a raw string.
func UnmarshalTaskFromString(data string, defaultWorkDir string, network string, envs []string, creds []*Credential) (*Task, error) {
	t, err := NewTaskFromString(data)
	if err != nil {
		return t, errors.Wrap(err, "failed to deserialize task and validate")
	}
	if defaultWorkDir != "" && t.WorkingDirectory == "" {
		t.WorkingDirectory = defaultWorkDir
	}

	t.Envs = envs
	t.Credentials = creds

	// External network parsed in from CLI will be set as default network, it will be used for any step if no network provide for them
	// The external network is append at the end of the list of networks, later we will do reverse iteration to get this network
	if network != "" {
		var externalNetwork *Network
		externalNetwork, err = NewNetwork(network, false, "external", true, true)
		if err != nil {
			return t, err
		}
		t.Networks = append(t.Networks, externalNetwork)
	}

	err = t.initialize()
	return t, err
}

// UnmarshalTaskFromFile unmarshals a Task from a file.
func UnmarshalTaskFromFile(file string, creds []*Credential) (*Task, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	t, err := NewTaskFromBytes(data)
	if err != nil {
		return t, errors.Wrap(err, "failed to deserialize task and validate")
	}

	t.Credentials = creds
	err = t.initialize()
	return t, err
}

// NewTaskFromString unmarshals a Task from string without any initialization.
func NewTaskFromString(data string) (*Task, error) {
	return NewTaskFromBytes([]byte(data))
}

// NewTaskFromBytes unmarshals a Task from given bytes without any initialization.
func NewTaskFromBytes(data []byte) (*Task, error) {
	t := &Task{}
	if err := yaml.Unmarshal(data, t); err != nil {
		return t, err
	}
	return t, t.Validate()
}

// Validate validates the task and returns an error if the Task has problems.
func (t *Task) Validate() error {
	// Validate secrets if exists
	idMap := make(map[string]struct{}, len(t.Secrets))
	for _, secret := range t.Secrets {
		err := secret.Validate()
		if err != nil {
			if secret.ID == "" {
				return err
			}
			return errors.Wrap(err, fmt.Sprintf("failed to validate secret with ID: %s", secret.ID))
		}

		if _, exists := idMap[secret.ID]; exists {
			return fmt.Errorf("duplicate secret found with ID: %s", secret.ID)
		}

		idMap[secret.ID] = struct{}{}
	}
	return nil
}

// NewTask returns a default Task object.
func NewTask(
	steps []*Step,
	secrets []*Secret,
	registry string,
	credentials []*Credential,
	totalTimeout int,
	isBuildTask bool) (*Task, error) {
	t := &Task{
		Steps:        steps,
		StepTimeout:  defaultStepTimeoutInSeconds,
		TotalTimeout: totalTimeout,
		Secrets:      secrets,
		RegistryName: registry,
		Credentials:  credentials,
		IsBuildTask:  isBuildTask,
	}

	err := t.initialize()
	return t, err
}

// initialize normalizes a Task's values.
func (t *Task) initialize() error {
	newDefaultNetworkName := DefaultNetworkName
	addDefaultNetworkToSteps := false

	// Reverse iterate the list to get the default network
	for i := len(t.Networks) - 1; i >= 0; i-- {
		network := t.Networks[i]
		if network.IsDefault {
			newDefaultNetworkName = network.Name
			addDefaultNetworkToSteps = true
			break
		}
	}

	// Add the default network if none are specified.
	// Only add the default network if we're using tasks.
	if !t.IsBuildTask && len(t.Networks) == 0 {
		defaultNetwork, err := NewNetwork(newDefaultNetworkName, false, "bridge", false, true)
		if err != nil {
			return err
		}
		if runtime.GOOS == windowsOS {
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

	for i, s := range t.Steps {
		// If individual steps don't have step timeouts specified,
		// stamp the global timeout on them.
		if s.Timeout <= 0 {
			s.Timeout = t.StepTimeout
		}

		if s.RetryDelayInSeconds <= 0 {
			s.RetryDelayInSeconds = defaultStepRetryDelayInSeconds
		}

		if addDefaultNetworkToSteps && s.Network == "" {
			s.Network = newDefaultNetworkName
		}

		newEnvs, err := mergeEnvs(s.Envs, t.Envs)
		if err != nil {
			return fmt.Errorf("bad format of environment variables, err: %v", err)
		}
		s.Envs = newEnvs

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
	return len(t.Credentials) > 0
}

// getNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func getNormalizedDockerImageNames(dockerImages []string, registry string) []string {
	if len(dockerImages) == 0 {
		return dockerImages
	}

	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, dockerImage := range dockerImages {
		d := scan.NormalizeImageTag(dockerImage)
		d = util.PrefixRegistryToImageName(registry, d)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}

// mergeEnvs merges the step's environment variables, overriding the task's default ones if provided.
func mergeEnvs(stepEnvs []string, taskEnvs []string) ([]string, error) {
	if len(taskEnvs) < 1 {
		return stepEnvs, nil
	}

	// preprocess the comma case
	var newTaskEnvs []string
	for _, env := range taskEnvs {
		newEnv := strings.Split(env, ",")
		newTaskEnvs = append(newTaskEnvs, newEnv...)
	}

	var stepmap = make(map[string]string)
	// parse stepEnvs into a map
	for _, env := range stepEnvs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			err := fmt.Errorf("cannot parse step environment variable %s correctly", env)
			return stepEnvs, err
		}
		stepmap[pair[0]] = pair[1]
	}

	// merge the unique taskEnvs into stepEnvs
	for _, env := range newTaskEnvs {
		pair := strings.SplitN(env, "=", 2)

		if len(pair) != 2 {
			err := fmt.Errorf("cannot parse task environment variable %s correctly", env)
			return stepEnvs, err
		}

		// if the env has not been provided, add to step env
		if _, ok := stepmap[pair[0]]; !ok {
			stepEnvs = append(stepEnvs, pair[0]+"="+pair[1])
		}

	}

	return stepEnvs, nil
}
