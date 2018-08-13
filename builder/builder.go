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

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	scannerutil "github.com/Azure/acr-builder/baseimages/scanner/util"
	"github.com/Azure/acr-builder/graph"
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
	if !b.procManager.DryRun {
		log.Printf("Setting up Docker configuration...")
		if err := b.setupConfig(ctx); err != nil {
			return err
		}
		log.Printf("Successfully set up Docker configuration")
		if task.UsingRegistryCreds() {
			log.Printf("Logging in to registry: %s\n", task.RegistryName)
			if err := b.dockerLoginWithRetries(ctx, task.RegistryName, task.RegistryUsername, task.RegistryPassword, 0); err != nil {
				return err
			}
			log.Println("Successfully logged in")
		}
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

	if err := b.pushWithRetries(ctx, task.Push); err != nil {
		return err
	}

	var deps []*models.ImageDependencies
	for _, n := range task.Dag.Nodes {
		step := n.Value
		log.Printf("Step ID %v marked as %v (elapsed time in seconds: %f)\n", step.ID, step.StepStatus, step.EndTime.Sub(step.StartTime).Seconds())

		if len(step.ImageDependencies) > 0 {
			if err := b.populateDigests(ctx, step.ImageDependencies); err != nil {
				return err
			}
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
		// TODO: Optimization - only execute on nodes which have been initiated.
		step := n.Value
		killArgs := append(args, step.ID)
		_ = b.procManager.Run(ctx, killArgs, nil, nil, nil, "")
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
		if err := b.runStep(ctx, step); err != nil {
			step.StepStatus = graph.Failed
			errorChan <- errors.Wrapf(err, "failed to run step id: %s", step.ID)
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
	log.Printf("Executing step: %s\n", step.ID)
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
		dockerfile, context := parseDockerBuildCmd(step.Build)
		volName := b.workspaceDir

		deps, err := b.scrapeDependencies(ctx, volName, step.WorkDir, step.ID, dockerfile, context, step.Tags, step.BuildArgs)
		if err != nil {
			return errors.Wrap(err, "failed to scan dependencies")
		}
		step.ImageDependencies = deps

		workDir := step.WorkDir
		// Modify the Run command if it's a tar or a git URL.
		if !scannerutil.IsLocalContext(context) {
			// NB: use step.ID as the working directory if the context is remote,
			// since we obtained the source code from the scanner and put it in this location.
			// If the remote context also has additional context specified, we have to append it
			// to the working directory.
			if scannerutil.IsGitURL(context) || scannerutil.IsVstsGitURL(context) {
				workDir = step.ID + "/" + getContextFromGitURL(context)
			} else {
				workDir = step.ID
			}

			step.Build = replacePositionalContext(step.Build, ".")
		}

		args = b.getDockerRunArgs(volName, workDir, step)
		args = append(args, "docker", "build")
		args = append(args, strings.Fields(step.Build)...)
	} else {
		args = b.getDockerRunArgs(b.workspaceDir, step.WorkDir, step)
		for _, env := range step.Envs {
			args = append(args, "--env", env)
		}
		if step.EntryPoint != "" {
			args = append(args, "--entrypoint", step.EntryPoint)
		}
		args = append(args, strings.Fields(step.Cmd)...)
	}

	if b.debug {
		log.Printf("Step args: %v\n", args)
	}
	// NB: ctx refers to the global ctx here,
	// so when running individual processes use the individual
	// step's ctx instead.
	timeout := time.Duration(step.Timeout) * time.Second
	currCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return b.procManager.Run(currCtx, args, nil, os.Stdout, os.Stderr, "")
}

// populateDigests populates digests on dependencies
func (b *Builder) populateDigests(ctx context.Context, dependencies []*models.ImageDependencies) error {
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

func (b *Builder) queryDigest(ctx context.Context, reference *models.ImageReference) error {
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
			"--volume", util.GetDockerSock(),
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

func getRepoDigest(jsonContent string, reference *models.ImageReference) string {
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
