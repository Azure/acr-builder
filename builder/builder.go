// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

// Builder builds images.
type Builder struct {
	procManager  *procmanager.ProcManager
	workspaceDir string
	debug        bool
}

// NewBuilder creates a new Builder.
func NewBuilder(pm *procmanager.ProcManager, debug bool, workspaceDir string) *Builder {
	return &Builder{
		procManager:  pm,
		debug:        debug,
		workspaceDir: workspaceDir,
	}
}

// RunTask executes a Task.
func (b *Builder) RunTask(ctx context.Context, task *graph.Task) error {
	for _, network := range task.Networks {
		if network.SkipCreation {
			log.Printf("Skip creating network: %s\n", network.Name)
			continue
		}
		log.Printf("Creating Docker network: %s, driver: '%s'\n", network.Name, network.Driver)
		if msg, err := network.Create(ctx, b.procManager); err != nil {
			return fmt.Errorf("Failed to create network: %s, err: %v, msg: %s", network.Name, err, msg)
		}
		log.Printf("Successfully set up Docker network: %s\n", network.Name)
	}

	log.Println("Setting up Docker configuration...")
	timeout := time.Duration(configTimeoutInSec) * time.Second
	configCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := b.setupConfig(configCtx); err != nil {
		return err
	}
	log.Printf("Successfully set up Docker configuration")
	if task.UsingRegistryCreds() {
		log.Printf("Logging in to registry: %s\n", task.RegistryName)
		timeout := time.Duration(loginTimeoutInSec) * time.Second
		loginCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := b.dockerLoginWithRetries(loginCtx, task.RegistryName, task.RegistryUsername, task.RegistryPassword, 0); err != nil {
			return err
		}
		log.Println("Successfully logged in")
	}

	var completedChans []chan bool
	errorChan := make(chan error)
	for _, n := range task.Dag.Nodes {
		completedChans = append(completedChans, n.Value.CompletedChan)
	}

	for _, n := range task.Dag.Root.Children() {
		go b.processVertex(ctx, task, task.Dag.Root, n, errorChan)
	}

	// Block until either:
	// - The global context expires
	// - A step has an error
	// - All steps have been processed
	for _, ch := range completedChans {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
			continue
		case err := <-errorChan:
			return err
		}
	}

	var deps []*image.Dependencies
	for _, step := range task.Steps {
		log.Printf("Step ID: %v marked as %v (elapsed time in seconds: %f)\n", step.ID, step.StepStatus, step.EndTime.Sub(step.StartTime).Seconds())

		if len(step.ImageDependencies) > 0 {
			log.Printf("Populating digests for step ID: %s...\n", step.ID)
			timeout := time.Duration(digestsTimeoutInSec) * time.Second
			digestCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			if err := b.populateDigests(digestCtx, step.ImageDependencies); err != nil {
				return err
			}
			log.Printf("Successfully populated digests for step ID: %s\n", step.ID)
			deps = append(deps, step.ImageDependencies...)
		}
	}

	if len(deps) > 0 {
		bytes, err := json.Marshal(deps)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal image dependencies")
		}
		fmt.Println("The following dependencies were found:")
		fmt.Println(string(bytes))
	}

	return nil
}

// CleanTask iterates through all build steps and removes
// their corresponding containers.
func (b *Builder) CleanTask(ctx context.Context, task *graph.Task) {
	args := []string{"docker", "rm", "-f"}
	for _, n := range task.Dag.Nodes {
		step := n.Value
		if step.StepStatus != graph.Skipped {
			killArgs := append(args, step.ID)
			_ = b.procManager.Run(ctx, killArgs, nil, nil, nil, "")
		}
	}

	for _, network := range task.Networks {
		if network.SkipCreation {
			log.Printf("Skip deleting network: %s\n", network.Name)
			continue
		}
		if msg, err := network.Delete(ctx, b.procManager); err != nil {
			log.Printf("Failed to delete network: %s, err: %v, msg: %s\n", network.Name, err, msg)
		}
	}

	_ = b.procManager.Stop()
}

func (b *Builder) processVertex(ctx context.Context, task *graph.Task, parent *graph.Node, child *graph.Node, errorChan chan error) {
	err := task.Dag.RemoveEdge(parent.Name, child.Name)
	if err != nil {
		errorChan <- errors.Wrap(err, "failed to remove edge")
		return
	}

	degree := child.GetDegree()
	if degree == 0 {
		step := child.Value
		err := b.runStep(ctx, step)
		if err != nil && step.IgnoreErrors {
			log.Printf("Step ID: %s encountered an error: %v, but is set to ignore errors. Continuing...\n", step.ID, err)
			step.StepStatus = graph.Successful
			for _, c := range child.Children() {
				go b.processVertex(ctx, task, child, c, errorChan)
			}
		} else if err != nil {
			step.StepStatus = graph.Failed
			errorChan <- errors.Wrapf(err, "failed to run step ID: %s", step.ID)
		} else {
			step.StepStatus = graph.Successful
			for _, c := range child.Children() {
				go b.processVertex(ctx, task, child, c, errorChan)
			}
		}
		// Step must always be marked as complete.
		step.CompletedChan <- true
	}
}

func (b *Builder) runStep(ctx context.Context, step *graph.Step) error {
	log.Printf("Executing step ID: %s. Working directory: '%s', Network: '%s'\n", step.ID, step.WorkingDirectory, step.Network)
	if step.StartDelay > 0 {
		log.Printf("Waiting %d seconds before executing step ID: %s\n", step.StartDelay, step.ID)
		time.Sleep(time.Duration(step.StartDelay) * time.Second)
	}

	step.StepStatus = graph.InProgress
	step.StartTime = time.Now()
	defer func() {
		step.EndTime = time.Now()
	}()

	var args []string

	if step.IsBuildStep() {
		dockerfile, dockerContext := parseDockerBuildCmd(step.Build)
		volName := b.workspaceDir

		validateDockerContext(dockerContext)

		log.Println("Obtaining source code and scanning for dependencies...")
		timeout := time.Duration(scrapeTimeoutInSec) * time.Second
		scrapeCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		deps, err := b.scrapeDependencies(scrapeCtx, volName, step.WorkingDirectory, step.ID, dockerfile, dockerContext, step.Tags, step.BuildArgs)
		if err != nil {
			return errors.Wrap(err, "failed to scan dependencies")
		}
		log.Println("Successfully obtained source code and scanned for dependencies")
		step.ImageDependencies = deps

		workingDirectory := step.WorkingDirectory
		// Modify the Run command if it's a tar or a git URL.
		if !util.IsLocalContext(dockerContext) {
			// NB: use step.ID as the working directory if the context is remote,
			// since we obtained the source code from the scanner and put it in this location.
			// If the remote context also has additional context specified, we have to append it
			// to the working directory.
			if util.IsSourceControlURL(dockerContext) {
				workingDirectory = step.ID + "/" + getContextFromGitURL(dockerContext)
			} else {
				workingDirectory = step.ID
			}
			step.Build = replacePositionalContext(step.Build, ".")
		}
		step.UpdateBuildStepWithDefaults()
		args = b.getDockerRunArgs(volName, workingDirectory, step, nil, "", "docker build "+step.Build)
	} else if step.IsPushStep() {
		timeout := time.Duration(step.Timeout) * time.Second
		pushCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return b.pushWithRetries(pushCtx, step.Push)
	} else {
		args = b.getDockerRunArgs(b.workspaceDir, step.WorkingDirectory, step, step.Envs, step.EntryPoint, step.Cmd)
	}

	if b.debug {
		log.Printf("Step args: %v\n", strings.Join(args, ", "))
	}

	timeout := time.Duration(step.Timeout) * time.Second
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return b.procManager.RunWithRetries(stepCtx, args, nil, os.Stdout, os.Stderr, "", step.Retries, step.RetryDelayInSeconds)
}

// populateDigests populates digests on dependencies
func (b *Builder) populateDigests(ctx context.Context, dependencies []*image.Dependencies) error {
	for _, entry := range dependencies {
		if err := b.queryDigest(ctx, entry.Image); err != nil {
			return err
		}
		if err := b.queryDigest(ctx, entry.Runtime); err != nil {
			return err
		}
		for _, buildtime := range entry.Buildtime {
			if err := b.queryDigest(ctx, buildtime); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) queryDigest(ctx context.Context, reference *image.Reference) error {
	if reference != nil {
		// refString will always have the tag specified at this point.
		// For "scratch", we have to compare it against "scratch:latest" even though
		// scratch:latest isn't valid in a FROM clause.
		if reference.Reference == NoBaseImageSpecifierLatest {
			return nil
		}
		args := []string{
			"docker",
			"run",
			"--rm",

			// Mount home
			"--volume", util.DockerSocketVolumeMapping,
			"--volume", homeVol + ":" + homeWorkDir,
			"--env", homeEnv,

			"docker",
			"inspect",
			"--format",
			"\"{{json .RepoDigests}}\"",
			reference.Reference,
		}
		if b.debug {
			log.Printf("query digest args: %v\n", args)
		}
		var buf bytes.Buffer
		if err := b.procManager.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
			return errors.Wrapf(err, "failed to query digests, msg: %s", buf.String())
		}
		trimCharPredicate := func(c rune) bool {
			return '\n' == c || '\r' == c || '"' == c || '\t' == c
		}
		reference.Digest = getRepoDigest(strings.TrimFunc(buf.String(), trimCharPredicate), reference)
	}

	return nil
}

func getRepoDigest(jsonContent string, reference *image.Reference) string {
	prefix := reference.Repository + "@"
	// If the reference has "library/" prefixed, we have to remove it - otherwise
	// we'll fail to query the digest, since image names aren't prefixed with "library/"
	if strings.HasPrefix(prefix, "library/") && reference.Registry == DockerHubRegistry {
		prefix = prefix[8:]
	} else if len(reference.Registry) > 0 && reference.Registry != DockerHubRegistry {
		prefix = reference.Registry + "/" + prefix
	}
	var digestList []string
	if err := json.Unmarshal([]byte(jsonContent), &digestList); err != nil {
		log.Printf("Error deserializing %s to json, error: %v\n", jsonContent, err)
	}
	for _, digest := range digestList {
		if strings.HasPrefix(digest, prefix) {
			return digest[len(prefix):]
		}
	}
	return ""
}

func validateDockerContext(context string) {
	if strings.Contains(context, "github") && !strings.Contains(context, ".git") {
		log.Printf("WARNING: %s might not be valid context. Valid Git repositories should end with .git.\n", context)
	}
}
