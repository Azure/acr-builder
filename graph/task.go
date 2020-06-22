// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strings"

	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/scan"
	"github.com/Azure/acr-builder/secretmgmt"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	// The default step timeout is 10 minutes.
	defaultStepTimeoutInSeconds = 60 * 10

	// The default step retry delay is 5 seconds.
	defaultStepRetryDelayInSeconds = 5

	// currentTaskVersion is the most recent Task version
	currentTaskVersion = "v1.0.0"

	noTaskNamePlaceholder = "quickrun"
)

var (
	validTaskVersions = map[string]bool{
		"1.0-preview-1":    true,
		currentTaskVersion: true,
		"v1.1.0":           true,
	}
)

// ResolvedRegistryCred is a credential with resolved username/password for the registry
type ResolvedRegistryCred struct {
	Username *secretmgmt.Secret
	Password *secretmgmt.Secret
}

// RegistryLoginCredentials is a map of registryName -> ResolvedRegistryCred
type RegistryLoginCredentials map[string]*ResolvedRegistryCred

// Task represents a task execution.
type Task struct {
	Steps                    []*Step               `yaml:"steps"`
	StepTimeout              int                   `yaml:"stepTimeout,omitempty"`
	Secrets                  []*secretmgmt.Secret  `yaml:"secrets,omitempty"`
	Networks                 []*Network            `yaml:"networks,omitempty"`
	VolumeMounts             []*volume.VolumeMount `yaml:"volumes,omitempty"`
	Envs                     []string              `yaml:"env,omitempty"`
	WorkingDirectory         string                `yaml:"workingDirectory,omitempty"`
	Version                  string                `yaml:"version,omitempty"`
	RegistryName             string
	Registry                 string
	TaskName                 string // Used to form the build cache image tag.
	Credentials              []*RegistryCredential
	RegistryLoginCredentials RegistryLoginCredentials
	Dag                      *Dag
	IsBuildTask              bool // Used to skip the default network creation for build.
	InitBuildkitContainer    bool // Used to initialize buildkit container if a build step is using build cache.
}

// TaskOptions are used to configure a new Task
type TaskOptions struct {
	// DefaultWorkingDir is the default working directory for the Task
	DefaultWorkingDir string

	// Network is the network the Task runs on
	Network string

	// Envs is a list of Task environment variables
	Envs []string

	// Credentials is a list of Registry credentials required to run a Task
	Credentials []*RegistryCredential

	// TaskName is the name of the Task
	TaskName string

	// Registry is the login server for Task
	Registry string

	// DoPreprocessing is a property to decide whether we use Alias
	DoPreprocessing bool

	// GlobalAliases keeps track of all the Task native global aliases
	GlobalAliases []byte
}

// UnmarshalTaskFromString unmarshals a Task from a raw string.
func UnmarshalTaskFromString(ctx context.Context, data string, opts *TaskOptions) (*Task, error) {
	t, err := NewTaskFromString(data)
	if err != nil {
		return t, errors.Wrap(err, "failed to deserialize task and validate")
	}
	err = t.AddTaskDefaults(ctx, opts)
	if err != nil {
		return t, err
	}
	return t, err
}

// AddTaskDefaults prepares a Task with remaining parameters
func (t *Task) AddTaskDefaults(ctx context.Context, opts *TaskOptions) error {
	if opts.DefaultWorkingDir != "" && t.WorkingDirectory == "" {
		t.WorkingDirectory = opts.DefaultWorkingDir
	}

	// Merge in the defaults with the Task's specific environment variables.
	// NB: Order is important here. Allow the Task's environment variables to override the defaults provided.
	newEnvs, err := mergeEnvs(t.Envs, opts.Envs)
	if err != nil {
		return err
	}

	t.Envs = newEnvs
	t.Credentials = opts.Credentials
	if opts.TaskName != "" {
		t.TaskName = opts.TaskName
	} else {
		t.TaskName = noTaskNamePlaceholder
	}

	t.Registry = opts.Registry

	// External network parsed in from CLI will be set as default network, it will be used for any step if no network provide for them
	// The external network is append at the end of the list of networks, later we will do reverse iteration to get this network
	if opts.Network != "" {
		var externalNetwork *Network
		externalNetwork, err = NewNetwork(opts.Network, false, "external", true, true)
		if err != nil {
			return err
		}
		t.Networks = append(t.Networks, externalNetwork)
	}

	err = t.initialize(ctx)
	return err
}

// UnmarshalTaskFromFile unmarshals a Task from a file.
func UnmarshalTaskFromFile(ctx context.Context, file string, opts *TaskOptions) (*Task, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	t, err := NewTaskFromBytes(data)
	if err != nil {
		return t, errors.Wrap(err, "failed to deserialize task and validate")
	}

	t.Credentials = opts.Credentials
	if opts.TaskName != "" {
		t.TaskName = opts.TaskName
	} else {
		t.TaskName = noTaskNamePlaceholder
	}
	err = t.initialize(ctx)
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

	//Validate Volumes if exists
	if err := ValidateVolumeMounts(t.VolumeMounts); err != nil {
		return err
	}
	//Validate that mounts reference a volume that exists
	for _, s := range t.Steps {
		if err := s.ValidateMountVolumeNames(t.VolumeMounts); err != nil {
			return err
		}
	}
	return nil
}

// NewTask returns a default Task object.
func NewTask(
	ctx context.Context,
	steps []*Step,
	secrets []*secretmgmt.Secret,
	registry string,
	credentials []*RegistryCredential,
	isBuildTask bool,
	defaultWorkDir string,
	taskName string) (*Task, error) {
	t := &Task{
		Steps:        steps,
		StepTimeout:  defaultStepTimeoutInSeconds,
		Secrets:      secrets,
		RegistryName: registry,
		Credentials:  credentials,
		IsBuildTask:  isBuildTask,
		TaskName:     taskName,
	}
	if defaultWorkDir != "" && t.WorkingDirectory == "" {
		t.WorkingDirectory = defaultWorkDir
	}
	if taskName != "" {
		t.TaskName = taskName
	} else {
		t.TaskName = noTaskNamePlaceholder
	}
	err := t.initialize(ctx)
	return t, err
}

// initialize normalizes a Task's values.
func (t *Task) initialize(ctx context.Context) error {
	newDefaultNetworkName := DefaultNetworkName
	addDefaultNetworkToSteps := false

	// Default the Task's to the latest version if it's unspecified.
	if t.Version == "" {
		t.Version = currentTaskVersion
	}
	if err := validateTaskVersion(t.Version); err != nil {
		return err
	}

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
		if runtime.GOOS == util.WindowsOS {
			defaultNetwork.Driver = "nat"
		}
		t.Networks = append(t.Networks, defaultNetwork)
		addDefaultNetworkToSteps = true
	}

	if t.StepTimeout <= 0 {
		t.StepTimeout = defaultStepTimeoutInSeconds
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
			return errors.Wrap(err, "failed to merge task and step environment variables")
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
			if len(s.Tags) == 0 {
				s.Tags = util.ParseTags(s.Build)
			}
			s.BuildArgs = util.ParseBuildArgs(s.Build)

			if s.UseBuildCacheForBuildStep() {
				if runtime.GOOS == util.LinuxOS {
					if buildStepWithBuildCache, err := s.GetCmdWithCacheFlags(t.TaskName, t.Registry); err != nil {
						log.Printf("error creating build cache command %v\n", err)
					} else {
						// update the Build cmd with buildx cache flags
						s.Build = buildStepWithBuildCache
						t.InitBuildkitContainer = true
					}
				} else {
					log.Println("build cache is not supported on windows. Will use standard docker build")
				}
			}
		} else if s.IsPushStep() {
			s.Push = getNormalizedDockerImageNames(s.Push)
		}
	}
	var err error

	t.RegistryLoginCredentials, err = ResolveCustomRegistryCredentials(ctx, t.Credentials)
	if err != nil {
		return err
	}
	t.Dag, err = NewDagFromTask(t)
	return err
}

// UsingRegistryCreds determines whether or not the Task is using registry creds.
func (t *Task) UsingRegistryCreds() bool {
	return len(t.RegistryLoginCredentials) > 0
}

// getNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func getNormalizedDockerImageNames(dockerImages []string) []string {
	if len(dockerImages) == 0 {
		return dockerImages
	}

	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, dockerImage := range dockerImages {
		d := scan.NormalizeImageTag(dockerImage)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}

// mergeEnvs merges the src environment variables into dest.
func mergeEnvs(dest []string, src []string) ([]string, error) {
	if len(src) < 1 {
		return dest, nil
	}

	var newEnvs []string
	for _, env := range src {
		newEnv := strings.Split(env, ",")
		newEnvs = append(newEnvs, newEnv...)
	}

	var stepmap = make(map[string]string)
	for _, env := range dest {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			err := fmt.Errorf("cannot parse step environment variable %s correctly", env)
			return dest, err
		}
		stepmap[pair[0]] = pair[1]
	}

	for _, env := range newEnvs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			err := fmt.Errorf("cannot parse task environment variable %s correctly", env)
			return dest, err
		}
		if _, ok := stepmap[pair[0]]; !ok {
			dest = append(dest, pair[0]+"="+pair[1])
		}
	}

	return dest, nil
}

// validateTaskVersion validates the specified version and returns an error if it isn't valid.
func validateTaskVersion(version string) error {
	vLower := strings.ToLower(version)
	if _, ok := validTaskVersions[vLower]; !ok {
		return fmt.Errorf("invalid version specified: %q, the current version is %q", version, currentTaskVersion)
	}
	return nil
}

// ResolveCustomRegistryCredentials resolves all the registry login credentials
func ResolveCustomRegistryCredentials(ctx context.Context, credentials []*RegistryCredential) (RegistryLoginCredentials, error) {
	resolvedCreds := make(RegistryLoginCredentials)
	var unresolvedCreds []*secretmgmt.Secret

	for _, cred := range credentials {
		if cred == nil {
			continue
		}
		resolvedCreds[cred.Registry] = &ResolvedRegistryCred{
			Username: &secretmgmt.Secret{
				ID: cred.Registry,
			},
			Password: &secretmgmt.Secret{
				ID: cred.Registry,
			},
		}
		isMSI := false

		usernameSecretObject := resolvedCreds[cred.Registry].Username
		passwordSecretObject := resolvedCreds[cred.Registry].Password

		switch cred.UsernameType {
		case Opaque:
			usernameSecretObject.ResolvedValue = cred.Username
		case VaultSecret:
			usernameSecretObject.KeyVault = cred.Username
			usernameSecretObject.MsiClientID = cred.Identity
			unresolvedCreds = append(unresolvedCreds, usernameSecretObject)
		case "":
			isMSI = true
		}

		switch cred.PasswordType {
		case Opaque:
			passwordSecretObject.ResolvedValue = cred.Password
		case VaultSecret:
			passwordSecretObject.KeyVault = cred.Password
			passwordSecretObject.MsiClientID = cred.Identity
			unresolvedCreds = append(unresolvedCreds, passwordSecretObject)
		}

		if isMSI {
			usernameSecretObject.ResolvedValue = "00000000-0000-0000-0000-000000000000"
			passwordSecretObject.MsiClientID = cred.Identity
			passwordSecretObject.AadResourceID = cred.AadResourceID
			unresolvedCreds = append(unresolvedCreds, passwordSecretObject)
		}
	}

	secretResolver, err := secretmgmt.NewSecretResolver(nil, secretmgmt.DefaultSecretResolveTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create secret resolver")
	}

	err = secretResolver.ResolveSecrets(ctx, unresolvedCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve secrets")
	}

	return resolvedCreds, nil
}

//ValidateVolumeMounts checks each volume is well formed and each container path is unique
func ValidateVolumeMounts(volMounts []*volume.VolumeMount) error {
	duplicate := make(map[string]struct{}, len(volMounts))
	for _, v := range volMounts {
		// call v.Validate() for each mount
		if err := v.Validate(); err != nil {
			return err
		}
		// make sure each volume name provided is unique
		if _, exists := duplicate[v.Name]; exists {
			return errors.New("volume with duplicate name found")
		}

		duplicate[v.Name] = struct{}{}
	}
	return nil
}
