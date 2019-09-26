// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/secretmgmt"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

const (
	buildTimeoutInSec = 60 * 60 * 8 // 8 hours
	pushTimeoutInSec  = 60 * 30     // 30 minutes
)

// TaskCreateOptions contains options to create a Task
type TaskCreateOptions struct {

	// TaskFile is a task file
	TaskFile string

	// Base64EncodedTaskFile
	Base64EncodedTaskFile string

	// WorkingDirectory
	WorkingDirectory string

	// Network
	Network string

	// Env
	Env []string

	// Credentials
	Credentials []string

	// ValuesFile
	ValuesFile string

	// Base64EncodedValuesFile
	Base64EncodedValuesFile string

	// SharedVolume
	SharedVolume string

	// ID
	ID string

	// Commit
	Commit string

	// Repository
	Repository string

	// Branch
	Branch string

	// TriggeredBy
	TriggeredBy string

	// GitTag
	GitTag string

	// Registry
	Registry string

	// OS
	OS string

	// OSVersion
	OSVersion string

	// Architecture
	Architecture string

	// SecretResolveTimeout
	SecretResolveTimeout time.Duration

	// TemplateValues
	TemplateValues []string

	// TaskName
	TaskName string

	// Isolation
	Isolation string

	// Labels
	Labels []string

	// Dockerfile
	Dockerfile string

	// Tags
	Tags []string

	// BuildArgs
	BuildArgs []string

	// SecretBuildArgs
	SecretBuildArgs []string

	// Target
	Target string

	// Platform
	Platform string

	// BuildContext
	BuildContext string

	// Date
	Date time.Time

	// Pull
	Pull bool

	// Push
	Push bool
	// NoCache
	NoCache bool
}

// CreateExecTask creates an Exec task
func CreateExecTask(ctx context.Context, opts *TaskCreateOptions) (*graph.Task, error) {
	renderOpts := &BaseRenderOptions{
		TaskFile:                opts.TaskFile,
		Base64EncodedTaskFile:   opts.Base64EncodedTaskFile,
		ValuesFile:              opts.ValuesFile,
		Base64EncodedValuesFile: opts.Base64EncodedValuesFile,
		TemplateValues:          opts.TemplateValues,
		ID:                      opts.ID,
		Commit:                  opts.Commit,
		Repository:              opts.Repository,
		Branch:                  opts.Branch,
		TriggeredBy:             opts.TriggeredBy,
		GitTag:                  opts.GitTag,
		Registry:                opts.Registry,
		Date:                    opts.Date,
		SharedVolume:            opts.SharedVolume,
		OS:                      opts.OS,
		OSVersion:               opts.OSVersion,
		Architecture:            opts.Architecture,
		SecretResolveTimeout:    opts.SecretResolveTimeout,
		TaskName:                opts.TaskName,
	}

	var template *Template
	var err error
	if opts.TaskFile != "" {
		if template, err = LoadTemplate(opts.TaskFile); err != nil {
			return nil, err
		}
	} else {
		if template, err = DecodeTemplate(opts.Base64EncodedTaskFile); err != nil {
			return nil, err
		}
	}

	var credentials []*graph.RegistryCredential

	if len(opts.Credentials) > 0 {
		// Add all creds provided by the user in the --credential flag
		credentials, err = graph.CreateRegistryCredentialFromList(opts.Credentials)
		if err != nil {
			return nil, errors.Wrap(err, "error creating registry credentials from given list")
		}
	}

	var task *graph.Task
	var alias *Alias
	versionInUse := FindVersion(template.GetData())
	shouldIncludeAlias := versionInUse >= "v1.1.0"
	if shouldIncludeAlias {
		log.Printf("Alias support enabled for version >= 1.1.0, please see https://aka.ms/acr/tasks/task-aliases for more information.")
		// separate alias and remaining data from the Task
		aliasData, taskData := SeparateAliasFromRest(template.GetData())

		// render alias data
		renderedAlias, renderAliasErr := LoadAndRenderSteps(ctx, NewTemplate("aliasData", aliasData), renderOpts)
		if renderAliasErr != nil {
			return nil, errors.Wrap(renderAliasErr, "unable to render alias data")
		}
		aliasData = []byte(renderedAlias)
		// Preprocess the task to replace all aliases based on the alias sources.
		processedTask, _alias, aliasErr := SearchReplaceAlias(template.GetData(), aliasData, taskData)
		alias = _alias
		if aliasErr != nil {
			return nil, errors.Wrap(renderAliasErr, "unable to search/replace aliases in task")
		}
		if isDebug(ctx) {
			log.Printf("Processed task before rendering data:\n%s", processedTask)
		}
		// update the template.Data
		template.Data = processedTask
	}

	rendered, err := LoadAndRenderSteps(ctx, template, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to render task")
	}
	if isDebug(ctx) {
		log.Printf("Rendered template:\n%s", rendered)
	}

	task, errUnmarshal := graph.UnmarshalTaskFromString(ctx, rendered, &graph.TaskOptions{
		DefaultWorkingDir: opts.WorkingDirectory,
		Network:           opts.Network,
		Envs:              opts.Env,
		Credentials:       credentials,
		TaskName:          opts.TaskName,
	})
	if errUnmarshal != nil {
		return nil, errors.Wrap(errUnmarshal, "failed to unmarshal task before running")
	}

	if shouldIncludeAlias {
		ExpandCommandAliases(alias, task)
	}

	return task, nil
}

// CreateBuildTask creates a Build task
func CreateBuildTask(ctx context.Context, opts *TaskCreateOptions) (*graph.Task, error) {
	renderOpts := &BaseRenderOptions{
		ValuesFile:              opts.ValuesFile,
		Base64EncodedValuesFile: opts.Base64EncodedValuesFile,
		TemplateValues:          opts.TemplateValues,
		ID:                      opts.ID,
		Commit:                  opts.Commit,
		Repository:              opts.Repository,
		Branch:                  opts.Branch,
		TriggeredBy:             opts.TriggeredBy,
		GitTag:                  opts.GitTag,
		Registry:                opts.Registry,
		Date:                    opts.Date,
		SharedVolume:            opts.SharedVolume,
		OS:                      opts.OS,
		OSVersion:               opts.OSVersion,
		Architecture:            opts.Architecture,
	}

	// Create the run command to be used in the template
	args := []string{}
	if opts.Isolation != "" {
		args = append(args, fmt.Sprintf("--isolation=%s", opts.Isolation))
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	for _, label := range opts.Labels {
		args = append(args, "--label", label)
	}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}
	for _, tag := range opts.Tags {
		args = append(args, "-t", tag)
	}
	for _, buildArg := range opts.BuildArgs {
		args = append(args, "--build-arg", buildArg)
	}
	for _, secretBuildArg := range opts.SecretBuildArgs {
		args = append(args, "--build-arg", secretBuildArg)
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}
	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}
	args = append(args, opts.BuildContext)
	runCmd := strings.Join(args, " ")

	// Create the template
	template := NewTemplate("build", []byte(runCmd))

	rendered, err := LoadAndRenderSteps(ctx, template, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if isDebug(ctx) {
		log.Println("Rendered template:")
		log.Println(rendered)
	}

	// After the template has rendered, we have to parse the tags again
	// so we can properly set the build/push tags.
	rendered, prefixedTags := util.PrefixTags(rendered, opts.Registry)
	opts.Tags = prefixedTags

	buildStep := &graph.Step{
		ID:      "build",
		Build:   rendered,
		Timeout: buildTimeoutInSec,
		Tags:    opts.Tags,
	}

	steps := []*graph.Step{buildStep}

	if opts.Push {
		pushStep := &graph.Step{
			ID:      "push",
			Push:    opts.Tags,
			Timeout: pushTimeoutInSec,
			When:    []string{buildStep.ID},
		}

		steps = append(steps, pushStep)
	}

	var credentials []*graph.RegistryCredential
	for _, credString := range opts.Credentials {
		cred, err := graph.CreateRegistryCredentialFromString(credString)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, cred)
	}

	return graph.NewTask(ctx, steps, []*secretmgmt.Secret{}, opts.Registry, credentials, true, opts.WorkingDirectory, "")
}

func isDebug(ctx context.Context) bool {
	debug := ctx.Value("debug")
	d, ok := debug.(bool)
	return ok && d
}
