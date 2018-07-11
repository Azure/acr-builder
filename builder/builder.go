// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/Azure/acr-builder/baseimages/scanner/util"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/taskmanager"
	builderutil "github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Builder builds images.
type Builder struct {
	taskManager  *taskmanager.TaskManager
	workspaceDir string
	mu           sync.Mutex
	debug        bool
}

// NewBuilder creates a new Builder.
func NewBuilder(tm *taskmanager.TaskManager, debug bool, workspaceDir string) *Builder {
	return &Builder{
		taskManager:  tm,
		debug:        debug,
		workspaceDir: workspaceDir,
		mu:           sync.Mutex{},
	}
}

// RunAllBuildSteps executes a pipeline.
func (b *Builder) RunAllBuildSteps(ctx context.Context, pipeline *graph.Pipeline) error {

	if !b.taskManager.DryRun {
		if err := b.setupConfig(ctx); err != nil {
			return err
		}

		if pipeline.UsingRegistryCreds() {
			fmt.Printf("Logging in to registry: %s\n", pipeline.RegistryName)
			if err := b.dockerLoginWithRetries(ctx, pipeline.RegistryName, pipeline.RegistryUsername, pipeline.RegistryPassword, 0); err != nil {
				return err
			}
			fmt.Println("Successfully logged in")
		}
	}

	root := pipeline.Dag.Nodes[graph.RootNodeID]
	var completedChans []chan bool
	errorChan := make(chan error)
	for k, v := range pipeline.Dag.Nodes {
		if k == graph.RootNodeID {
			continue
		}
		completedChans = append(completedChans, v.Value.CompletedChan)
	}

	for _, n := range root.Children() {
		go b.processVertex(ctx, pipeline, root, n, errorChan)
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

	if err := b.pushWithRetries(ctx, pipeline.Push); err != nil {
		return err
	}

	for k, v := range pipeline.Dag.Nodes {
		if k == graph.RootNodeID {
			continue
		}

		step := v.Value

		err := b.PopulateDigests(ctx, step.ImageDependencies)
		if err != nil {
			return errors.Wrap(err, "failed to populate digests")
		}

		fmt.Printf("Step ID %v marked as %v (elapsed time in seconds: %f)\n", step.ID, step.StepStatus, step.EndTime.Sub(step.StartTime).Seconds())
		bytes, err := json.Marshal(step.ImageDependencies)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal image dependencies")
		}

		fmt.Println(string(bytes))
	}

	return nil
}

// CleanAllBuildSteps cleans up all build steps and removes their corresponding Docker containers.
func (b *Builder) CleanAllBuildSteps(ctx context.Context, pipeline *graph.Pipeline) {
	args := []string{"docker", "rm", "-f"}

	for k, v := range pipeline.Dag.Nodes {
		if k == graph.RootNodeID {
			continue
		}

		step := v.Value

		killArgs := append(args, fmt.Sprintf("acb_step_%s", step.ID))
		_ = b.taskManager.Run(ctx, killArgs, nil, nil, nil, "")
	}

	if err := b.taskManager.Stop(); err != nil {
		fmt.Printf("Failed to stop ongoing processes: %v", err)
	}
}

func (b *Builder) processVertex(ctx context.Context, pipeline *graph.Pipeline, parent *graph.Node, child *graph.Node, errorChan chan error) {
	err := pipeline.Dag.RemoveEdge(parent.Name, child.Name)
	if err != nil {
		errorChan <- errors.Wrap(err, "failed to remove edge")
		return
	}

	degree := child.GetDegree()
	if degree == 0 {
		step := child.Value
		step.StepStatus = graph.InProgress
		step.StartTime = time.Now()
		defer func() {
			step.EndTime = time.Now()
		}()

		var args []string

		if strings.HasPrefix(step.Run, "build ") {
			dockerfile, context := parseDockerBuildCmd(step.Run)
			volName := b.workspaceDir
			stepWorkingDir := step.ID

			// If we run `build` (not `exec` for a pipeline) and the context is local,
			// we get the absolute filepath of the context they provided and volume map
			// the host's filesystem according to the context.
			if util.IsLocalContext(context) {
				if step.UseLocalContext {
					path, err := filepath.Abs(context)
					if err != nil {
						errorChan <- errors.Wrap(err, "failed to normalize local context")
						return
					}
					volName = filepath.Clean(path)

					// Other steps end up in an output directory which matches the step's ID,
					// but in this case there's no output directory.
					stepWorkingDir = ""
				} else {
					// A local pipeline re-uses a volume.
					stepWorkingDir = step.WorkDir
				}
			}

			deps, err := b.scrapeDependencies(ctx, volName, step.WorkDir, step.ID, dockerfile, context, step.Tags, step.BuildArgs)
			if err != nil {
				errorChan <- errors.Wrap(err, "failed to scan dependencies")
				return
			}
			step.ImageDependencies = deps

			// Modify the Run command if it's a tar or a git URL.
			if !util.IsLocalContext(context) {
				// Allow overriding the context from the git URL.
				if util.IsGitURL(context) {
					step.Run = replacePositionalContext(step.Run, getContextFromGitURL(context))
				} else {
					step.Run = replacePositionalContext(step.Run, ".")
				}
			}

			args = b.getDockerRunArgs(volName, step.ID, stepWorkingDir)
			args = append(args, "docker")
			args = append(args, strings.Fields(step.Run)...)

		} else {
			args = b.getDockerRunArgs(b.workspaceDir, step.ID, step.WorkDir)
			for _, env := range step.Envs {
				args = append(args, "--env", env)
			}
			if step.EntryPoint != "" {
				args = append(args, "--entrypoint", step.EntryPoint)
			}
			args = append(args, strings.Fields(step.Run)...)
		}

		if b.debug {
			fmt.Printf("Args: %v\n", args)
		}

		// TODO: secret envs

		// NB: ctx refers to the global ctx here,
		// so when running individual processes use the individual
		// step's ctx instead.
		timeout := time.Duration(step.Timeout) * time.Second
		currCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := b.taskManager.Run(currCtx, args, nil, os.Stdout, os.Stderr, ""); err != nil {
			step.StepStatus = graph.Failed
			errorChan <- err
		} else {
			step.StepStatus = graph.Successful

			// Try to process all child nodes
			for _, c := range child.Children() {
				go b.processVertex(ctx, pipeline, child, c, errorChan)
			}
		}

		step.CompletedChan <- true
	} else if b.debug {
		fmt.Printf("Skipped processing %v, degree: %v\n", child.Name, degree)
	}
}

// PopulateDigests populates digests on dependencies
func (b *Builder) PopulateDigests(ctx context.Context, dependencies []*models.ImageDependencies) error {
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
			"--volume", builderutil.GetDockerSock(),
			"--volume", homeVol + ":" + homeWorkDir,
			"--env", homeEnv,

			"docker",
			"inspect",
			"--format",
			"\"{{json .RepoDigests}}\"",
			reference.Reference,
		}

		if b.debug {
			fmt.Printf("query digest args: %v\n", args)
		}

		var buf bytes.Buffer
		err := b.taskManager.Run(ctx, args, nil, &buf, os.Stderr, "")
		if err != nil {
			return err
		}

		trimCharPredicate := func(c rune) bool {
			return '\n' == c || '\r' == c || '"' == c || '\t' == c
		}

		reference.Digest = getRepoDigest(strings.TrimFunc(buf.String(), trimCharPredicate), reference)
	}

	return nil
}

func getRepoDigest(jsonContent string, reference *models.ImageReference) string {
	// Input: ["docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], , docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: ["test.azurecr.io/docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], test.azurecr.io, docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: Invalid
	// Output: <empty>

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
		logrus.Warnf("Error deserializing %s to json, error: %s", jsonContent, err)
	}

	for _, digest := range digestList {
		if strings.HasPrefix(digest, prefix) {
			return digest[len(prefix):]
		}
	}

	logrus.Warnf("Unable to find digest for %s in %s", prefix, jsonContent)
	return ""
}
