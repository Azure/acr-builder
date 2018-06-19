package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/acr-builder/cmder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

const (
	containerWorkspaceDir = "/workspace"
	rmContainer           = true

	rallyHomeWorkDir = "/rally/home"
	rallyHomeVol     = "rallyHomeVol"
)

// Builder builds images.
type Builder struct {
	cmder        *cmder.Cmder
	workspaceDir string
	dryRun       bool
	mu           sync.Mutex
	debug        bool
	buildOptions *BuildOptions
}

// NewBuilder creates a new Builder.
func NewBuilder(c *cmder.Cmder, debug bool, workspaceDir string, dryRun bool, buildOptions *BuildOptions) *Builder {
	return &Builder{
		cmder:        c,
		debug:        debug,
		workspaceDir: workspaceDir,
		dryRun:       dryRun,
		buildOptions: buildOptions,
		mu:           sync.Mutex{},
	}
}

// RunAllBuildSteps executes a pipeline.
func (b *Builder) RunAllBuildSteps(ctx context.Context, dag *graph.Dag, pushTo []string) error {
	// TODO: DESIGN: do we want multiple volumes per step?
	root := dag.Nodes[graph.RootNodeID]
	var completedChans []chan bool
	errorChan := make(chan error)
	for k, v := range dag.Nodes {
		if k == graph.RootNodeID {
			continue
		}
		completedChans = append(completedChans, v.Value.CompletedChan)
	}

	for _, n := range root.Children() {
		go b.processVertex(ctx, dag, root, n, errorChan)
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

	if b.buildOptions.Push {
		if err := b.Push(ctx, pushTo); err != nil {
			return errors.Wrap(err, "failed to push images")
		}
	}

	for k, v := range dag.Nodes {
		if k == graph.RootNodeID {
			continue
		}

		step := v.Value

		err := b.PopulateDigests(ctx, step.ImageDependencies)
		if err != nil {
			fmt.Printf("failed to populate digests, err: %v\n", err)
		}

		fmt.Printf("Step ID %v marked as %v (start time: %v, end time: %v)\n", step.ID, step.StepStatus, step.StartTime, step.EndTime)
		bytes, err := json.Marshal(step.ImageDependencies)
		if err != nil {
			fmt.Printf("Err while unmarshaling image dependencies: %v\n", err)
		} else {
			fmt.Println(string(bytes))
		}
	}

	return nil
}

// CleanAllBuildSteps cleans up all build steps and removes their corresponding Docker containers.
func (b *Builder) CleanAllBuildSteps(ctx context.Context) {
	// TODO: implement

	// args := []string{"docker", "rm", "-f"}

	// errs := cmder.Stop()
	// 	if errs != nil && debug {
	// 		fmt.Printf("Err during cleanup: %v", errs.String())
	// 	}
}

func (b *Builder) processVertex(ctx context.Context, dag *graph.Dag, parent *graph.Node, child *graph.Node, errorChan chan error) {
	err := dag.RemoveEdge(parent.Name, child.Name)
	if err != nil {
		errorChan <- errors.Wrap(err, "failed to remove edge")
		return
	}

	// TODO: review how to refactor this and safely exit; write to CompletedChan?

	degree := child.GetDegree()
	if degree == 0 {
		step := child.Value
		step.StepStatus = graph.InProgress
		step.StartTime = time.Now()
		defer func() {
			step.EndTime = time.Now()
		}()

		args := b.getDockerRunArgs(step.ID, step.WorkDir)
		for _, env := range step.Envs {
			args = append(args, "--env", env)
		}
		if step.EntryPoint != "" {
			args = append(args, "--entrypoint", step.EntryPoint)
		}

		if strings.HasPrefix(step.Run, "build") {
			// TODO: consider other cases where we should login, e.g., if they want to pull an image from their local registry.
			// For now, only login if they specify push.
			if b.buildOptions.Push {
				err := b.dockerLoginWithRetries(ctx, 0)
				if err != nil {
					errorChan <- err
					return
				}
			}

			dockerfile, context := util.ParseDockerBuildCmd(step.Run)
			workingDir, sha, err := b.obtainSourceCode(ctx, context, dockerfile)
			if err != nil {
				errorChan <- errors.Wrap(err, "failed to obtain source code")
				return
			}

			// Adjust the run command so that the ACR registry is prefixed for all tags
			step.Run = prefixStepTags(step.Run, b.buildOptions.RegistryName)
			tags := parseRunArgs(step.Run, "-t")
			buildArgs := parseRunArgs(step.Run, "--build-arg")
			deps, err := b.ScanForDependencies(workingDir, dockerfile, buildArgs, tags)
			if err != nil {
				errorChan <- errors.Wrap(err, "failed to scan dependencies")
				return
			}
			for _, dep := range deps {
				dep.Git = &graph.GitReference{
					GitHeadRev: sha,
				}
			}
			step.ImageDependencies = deps

			// If the step has a context directory specified, copy the context.
			if step.ContextDir != "" {
				if err := b.copyContext(ctx, workingDir, step.ContextDir); err != nil {
					errorChan <- err
					return
				}
			}

			args = append(args, "docker")
			args = append(args, strings.Fields(step.Run)...)
		} else {
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
		if err := b.cmder.Run(currCtx, args, nil, os.Stdout, os.Stderr, ""); err != nil {
			step.StepStatus = graph.Failed
			errorChan <- err
		} else {
			step.StepStatus = graph.Successful

			// Try to process all child nodes
			for _, c := range child.Children() {
				go b.processVertex(ctx, dag, child, c, errorChan)
			}
		}

		step.CompletedChan <- true
	} else if b.debug {
		fmt.Printf("Skipped processing %v, degree: %v\n", child.Name, degree)
	}
}

func prefixStepTags(runCmd string, registry string) string {
	if registry == "" {
		return runCmd
	}

	fields := strings.Fields(runCmd)

	for i := 1; i < len(fields); i++ {
		if fields[i-1] == "-t" {
			fields[i] = prefixRegistryToImageName(registry, fields[i])
		}
	}

	return strings.Join(fields, " ")
}
