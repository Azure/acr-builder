// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/util"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

const (
	// ImmediateExecutionToken defines the when dependency to indicate a step should execute immediately.
	ImmediateExecutionToken = "-"
	enabled                 = "enabled"
	disabled                = "disabled"
)

var (
	errMissingID         = errors.New("step is missing an ID")
	errMissingProps      = errors.New("step is missing a cmd, build, or push property")
	errIDContainsSpace   = errors.New("step ID cannot contain spaces")
	errInvalidDeps       = errors.New("step cannot contain other IDs in when if the immediate execution token is specified")
	errInvalidStepType   = errors.New("step must only contain a single build, cmd, or push property")
	errInvalidRetries    = errors.New("step must specify retries >= 0")
	errInvalidRepeat     = errors.New("step must specify repeat >= 0")
	errInvalidCacheValue = errors.New("invalid value for cache property. Valid values are 'enabled', 'disabled'")
	errInvalidMountsUse  = errors.New("invalid use of Mounts. Mounts must have unique container paths and used for CMD steps")
)

type chanBool chan bool

// MarshalYAML for chan bool's in step. Avoids having Marshall try to render chan bool values as these
// cannot be marshalled. Removing this interface causes a crash when marshaling a task.
func (c chanBool) MarshalYAML() (interface{}, error) {
	return "", nil
}

// UnMarshalYAML for chan bool's in step. Avoids having UnMarshal try to resolve chan bool values as these
// cannot be unmarshaled. Removing this interface causes a crash when umarshaling a task.
func (c chanBool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return nil
}

// Step is a step in the execution task.
type Step struct {
	ID                  string          `yaml:"id"`
	Cmd                 string          `yaml:"cmd"`
	Build               string          `yaml:"build"`
	WorkingDirectory    string          `yaml:"workingDirectory"`
	EntryPoint          string          `yaml:"entryPoint"`
	User                string          `yaml:"user"`
	Network             string          `yaml:"network"`
	Isolation           string          `yaml:"isolation"`
	CPUS                string          `yaml:"cpus"`
	Cache               string          `yaml:"cache"`
	Mounts              []*volume.Mount `yaml:"volumeMounts"`
	Push                []string        `yaml:"push"`
	Envs                []string        `yaml:"env"`
	Expose              []string        `yaml:"expose"`
	Ports               []string        `yaml:"ports"`
	When                []string        `yaml:"when"`
	ExitedWith          []int           `yaml:"exitedWith"`
	ExitedWithout       []int           `yaml:"exitedWithout"`
	Timeout             int             `yaml:"timeout"`
	StartDelay          int             `yaml:"startDelay"`
	RetryDelayInSeconds int             `yaml:"retryDelay"`
	// Retries specifies how many times a Step will be retried if it fails after its initial execution.
	Retries       int      `yaml:"retries"`
	RetryOnErrors []string `yaml:"retryOnErrors"`
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
	CompletedChan chanBool

	ImageDependencies    []*image.Dependencies
	Tags                 []string
	BuildArgs            []string
	DefaultBuildCacheTag string
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
	if s.HasMounts() {
		if !s.IsCmdStep() {
			return errInvalidMountsUse
		}
		valMounts := ValidateMounts(s.Mounts)
		if valMounts != nil {
			return valMounts
		}
	}
	for _, dep := range s.When {
		if dep == ImmediateExecutionToken && len(s.When) > 1 {
			return errInvalidDeps
		}
		if dep == s.ID {
			return NewSelfReferencedStepError(fmt.Sprintf("Step ID: %v is self-referenced", s.ID))
		}
	}

	if s.Cache != "" && !strings.EqualFold(s.Cache, enabled) && !strings.EqualFold(s.Cache, disabled) {
		return errInvalidCacheValue
	}

	return nil
}

//ValidateMounts checks each mount is well formed and each container file path is unique
func ValidateMounts(mounts []*volume.Mount) error {
	duplicate := make(map[string]struct{}, len(mounts))
	for _, m := range mounts {
		// call m.Validate() for each mount
		if err := m.Validate(); err != nil {
			return err
		}
		// make sure each container file path provided is unique
		if _, exists := duplicate[m.MountPath]; exists {
			return errors.New("mount with duplicate container file path found")
		}

		duplicate[m.MountPath] = struct{}{}
	}
	return nil
}

//ValidateMountVolumeNames checks mount name matches a listed volume
func (s *Step) ValidateMountVolumeNames(vols []*volume.VolumeMount) error {
	//for each mount in the step, check to see that there exists a matching
	nameMap := make(map[string]struct{}, len(vols))
	for _, v := range vols {
		nameMap[v.Name] = struct{}{}
	}
	for _, m := range s.Mounts {
		if _, exists := nameMap[m.Name]; !exists {
			return errors.New("provided mount name does not correspond to a volume")
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
		s.Cache == t.Cache &&
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

// HasMounts returns true if the Step has at least 1 mount listed, false otherwise
func (s *Step) HasMounts() bool {
	if s == nil {
		return false
	}
	return len(s.Mounts) > 0
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
	if s.IsBuildStep() && runtime.GOOS == util.WindowsOS && !strings.Contains(s.Build, "--isolation") {
		s.Build = fmt.Sprintf("--isolation hyperv %s -m 2GB", s.Build)
	}
}

// UseBuildCacheForBuildStep indicates if buildx needs to be used.
func (s *Step) UseBuildCacheForBuildStep() bool {
	return s != nil && s.IsBuildStep() && strings.ToLower(s.Cache) == enabled
}

// GetBuildCacheImageTag returns a default cacheid used to tag buildx images.
func GetBuildCacheImageTag(taskName, stepID string) string {
	return fmt.Sprintf("cache_%s_%s", taskName, stepID)
}

// GetCmdWithCacheFlags adds buildx cache parameters to the cmd.
func (s *Step) GetCmdWithCacheFlags(taskName, registry string) (string, error) {
	var domain, path, firstTagPath string
	var err error

	if strings.ToLower(s.Cache) != enabled {
		return "", errors.New("cache needs to be set to 'enabled' to use build cache")
	}
	if len(s.Tags) == 0 {
		return s.Build, nil
	}

	for idx, tag := range s.Tags {
		domain, path, err = getDomainPath(tag)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse the tag into a domain and path")
		}
		if idx == 0 {
			firstTagPath = path
		}
		if domain != "" {
			break
		}
	}

	if domain == "" {
		domain = registry
		path = firstTagPath
	}

	s.DefaultBuildCacheTag = GetBuildCacheImageTag(taskName, s.ID)
	return addBuildCacheOptsToCmd(domain, path, s.DefaultBuildCacheTag, s.Build)
}

// getDomainPath gets the domain and path for an image repository
func getDomainPath(s string) (string, string, error) {
	repo, err := reference.Parse(s)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse the image name")
	}
	named, ok := repo.(reference.Named)
	if !ok {
		return "", "", errors.New("failed to extract the name from registry url")
	}
	d, p := reference.SplitHostname(reference.TrimNamed(named))
	return d, p, nil
}

// addBuildCacheOptsToCmd appends the build cache options to the original Build command
func addBuildCacheOptsToCmd(domain, path, tag, originalBuildCmd string) (string, error) {
	named, err := reference.WithName(domain + "/" + path)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse reference to be used for cache image")
	}
	cacheImage, err := reference.WithTag(named, tag)
	if err != nil {
		return "", errors.Wrap(err, "failed to attach cache ID tag to the repo for build cache")
	}
	return fmt.Sprintf("--load --cache-to=type=registry,ref=%s,mode=max --cache-from=type=registry,ref=%s %s", cacheImage.String(), cacheImage.String(), originalBuildCmd), nil
}
