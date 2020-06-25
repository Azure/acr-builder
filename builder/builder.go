// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

const (
	dockerImg           = "docker"
	buildxImg           = "buildx"
	nanoServerImageName = "mcr.microsoft.com/windows/nanoserver:2004"
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
			return fmt.Errorf("failed to create network: %s, err: %v, msg: %s", network.Name, err, msg)
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
	log.Println("Successfully set up Docker configuration")
	if task.UsingRegistryCreds() {
		timeout := time.Duration(loginTimeoutInSec) * time.Second
		for registry, cred := range task.RegistryLoginCredentials {
			loginCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			log.Printf("Logging in to registry: %s\n", registry)
			if err := b.dockerLoginWithRetries(loginCtx, registry, cred.Username.ResolvedValue, cred.Password.ResolvedValue, 0); err != nil {
				return err
			}
			log.Printf("Successfully logged into %s\n", registry)
		}
	}

	var completedChans []chan bool
	errorChan := make(chan error)
	for _, node := range task.Dag.Nodes {
		completedChans = append(completedChans, node.Value.CompletedChan)
	}

	if task.InitBuildkitContainer {
		log.Println("Task will use build cache, initializing buildkitd container")
		// --workdir = /workspace
		args := b.getDockerRunArgs(
			make(map[string]string),
			b.workspaceDir,
			"",
			false,
			true,
			true,
			[]string{},
			[]string{},
			[]string{},
			false,
			"",
			"",
			"",
			"",
			"",
			buildkitdContainerName,
			buildxImg+" create --use",
		)
		if b.debug {
			log.Printf("buildkitd container args: %v\n", strings.Join(args, ", "))
		}

		timeout := time.Duration(buildkitdContainerRunTimeoutInSeconds) * time.Second
		buildkitCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err := b.procManager.RunRepeatWithRetries(
			buildkitCtx,
			args,
			nil,
			os.Stdout,
			os.Stderr,
			"",
			buildkitdContainerInitRetries,
			nil,
			buildkitdContainerInitRetryDelay,
			buildkitdContainerName,
			buildkitdContainerInitRepeat,
			false)
		if err != nil {
			log.Printf("buildx create --use failed with error: '%v'", err)
		}
	}

	//For each volume:
	for _, volMount := range task.VolumeMounts {
		// 1. take the values and dump into a file
		if err := b.createFilesForVolume(ctx, volMount); err != nil {
			return err
		}
		// 2. create a dummy data container which makes new volume and puts file in it
		if err := b.populateVolumeWithFiles(ctx, volMount); err != nil {
			return err
		}
	}

	for _, child := range task.Dag.Root.Children() {
		go b.processVertex(ctx, task, task.Dag.Root, child, errorChan)
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
		// currently we have a limitation where we cannot fetch the digest of the dependencies
		// because we build the image using buildx, but query the digest using docker.
		// https://github.com/Azure/acr-builder/issues/522
		if step.UseBuildCacheForBuildStep() && runtime.GOOS == util.LinuxOS {
			continue
		}
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
		depBytes, err := json.Marshal(deps)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal image dependencies")
		}
		log.Println("The following dependencies were found:")
		log.Println("\n" + string(depBytes))
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
	log.Printf("Executing step ID: %s. Timeout(sec): %d, Working directory: '%s', Network: '%s'\n", step.ID, step.Timeout, step.WorkingDirectory, step.Network)
	if step.StartDelay > 0 {
		log.Printf("Waiting %d seconds before executing step ID: %s\n", step.StartDelay, step.ID)
		time.Sleep(time.Duration(step.StartDelay) * time.Second)
	}

	if step.IsCmdStep() && step.Pull {
		log.Printf("Step specified pull. Performing an explicit pull...\n")
		if err := b.pullImageBeforeRun(ctx, step.Cmd); err != nil {
			return err
		}
	}

	step.StepStatus = graph.InProgress
	step.StartTime = time.Now()
	defer func() {
		step.EndTime = time.Now()
	}()

	var args []string

	if step.IsBuildStep() {
		dockerfile, target, dockerContext := parseDockerBuildCmd(step.Build)
		volName := b.workspaceDir

		// Print out a warning message if a remote context doesn't appear to be valid, i.e. doesn't end with .git.
		validateDockerContext(dockerContext)

		log.Println("Scanning for dependencies...")
		timeout := time.Duration(scrapeTimeoutInSec) * time.Second
		scrapeCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		deps, err := b.scrapeDependencies(scrapeCtx, volName, step.WorkingDirectory, step.ID, dockerfile, dockerContext, step.Tags, step.BuildArgs, target)
		if err != nil {
			return errors.Wrap(err, "failed to scan dependencies")
		}
		log.Println("Successfully scanned dependencies")
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

		if step.UseBuildCacheForBuildStep() {
			args = b.getDockerRunArgsForStep(volName, workingDirectory, step, "", buildxImg+" build "+step.Build)
		} else {
			args = b.getDockerRunArgsForStep(volName, workingDirectory, step, "", dockerImg+" build "+step.Build)
		}
	} else if step.IsPushStep() {
		timeout := time.Duration(step.Timeout) * time.Second
		pushCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return b.pushWithRetries(pushCtx, step.Push)
	} else {
		args = b.getDockerRunArgsForStep(b.workspaceDir, step.WorkingDirectory, step, step.EntryPoint, step.Cmd)
	}

	if b.debug {
		log.Printf("Step args: %v\n", strings.Join(args, ", "))
	}

	timeout := time.Duration(step.Timeout) * time.Second
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return b.procManager.RunRepeatWithRetries(
		stepCtx,
		args,
		nil,
		os.Stdout,
		os.Stderr,
		"",
		step.Retries,
		step.RetryOnErrors,
		step.RetryDelayInSeconds,
		step.ID,
		step.Repeat,
		step.IgnoreErrors)
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
			return c == '\n' || c == '\r' || c == '"' || c == '\t'
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

func validateDockerContext(sourceContext string) {
	sourceContext = strings.ToLower(sourceContext)
	if strings.Contains(sourceContext, "github") && !strings.Contains(sourceContext, ".git") {
		log.Printf("WARNING: %s might not be valid context. Valid Git repositories should end with .git.\n", sourceContext)
	}
}

func (b *Builder) pullImageBeforeRun(ctx context.Context, cmdArgs string) error {
	imageName := parseImageNameFromArgs(cmdArgs)
	args := []string{
		"docker",
		"run",
		"--rm",
		"--volume", util.DockerSocketVolumeMapping,
		"docker",
		"pull",
		imageName,
	}
	if b.debug {
		log.Printf("pull image args: %v\n", args)
	}
	return b.procManager.Run(ctx, args, nil, os.Stdout, os.Stdout, "")
}

// parseImageNameFromArgs parses an image's name from a command step's arguments.
func parseImageNameFromArgs(cmdArgs string) string {
	idx := strings.Index(cmdArgs, " ")
	if idx < 0 {
		return cmdArgs
	}
	return cmdArgs[:idx]
}

func (b *Builder) createFilesForVolume(ctx context.Context, volMount *volume.VolumeMount) error {
	var args []string
	var sb strings.Builder
	if runtime.GOOS == util.WindowsOS {
		args = []string{"powershell.exe", "-Command"}
		sb.WriteString("mkdir " + volMount.Name)
		log.Println("making directory " + volMount.Name)
	} else {
		args = []string{"/bin/sh", "-c"}
		sb.WriteString("mkdir " + volMount.Name + " && ")
	}
	for _, values := range volMount.Values {
		for k, v := range values {
			val := v
			decoded, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return errors.New("failed to decode Base64 value. please make sure value provided is Base64 encoded")
			}
			val = string(decoded)
			if runtime.GOOS == util.WindowsOS {
				sb.WriteString("; Add-Content -Path ")
				sb.WriteString(volMount.Name + "/" + k)
				sb.WriteString(" -Value @\"\n")
				sb.WriteString(val)
				sb.WriteString("\n\"@")
			} else {
				sb.WriteString("cat >> ")
				sb.WriteString(volMount.Name + "/" + k)
				sb.WriteString(" <<EOL\n")
				sb.WriteString(val)
				sb.WriteString("\nEOL\n")
			}
		}
	}
	args = append(args, sb.String())
	var buf bytes.Buffer
	if err := b.procManager.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
		return errors.Wrapf(err, "failed to write value, %s", buf.String())
	}
	log.Println("buffer: ", buf.String())
	log.Println("Created new file(s) for volume: ", volMount.Name)
	return nil
}

func (b *Builder) populateVolumeWithFiles(ctx context.Context, volMount *volume.VolumeMount) error {
	var dataContainerArgs []string
	var dataSB strings.Builder
	if runtime.GOOS == util.WindowsOS {
		dataContainerArgs = []string{"powershell.exe", "-Command"}
		dataSB.WriteString("docker run --rm -v " + b.workspaceDir + "\\" + volMount.Name + ":c:\\source -v ")
		dataSB.WriteString(volMount.Name + ":c:\\dest -w /source ")
		dataSB.WriteString(nanoServerImageName + " cmd.exe /c copy c:\\source c:\\dest")
	} else {
		dataContainerArgs = []string{"/bin/sh", "-c"}
		dataSB.WriteString("docker run --rm -v " + b.workspaceDir + ":/source -v ")
		dataSB.WriteString(volMount.Name + ":/dest -w /source alpine cp ")
		for _, values := range volMount.Values {
			for k := range values {
				dataSB.WriteString(volMount.Name + "/" + k)
				dataSB.WriteString(" ")
			}
		}
		dataSB.WriteString("/dest")
	}
	dataContainerArgs = append(dataContainerArgs, dataSB.String())
	var buf bytes.Buffer
	if err := b.procManager.Run(ctx, dataContainerArgs, nil, &buf, &buf, ""); err != nil {
		return errors.Wrapf(err, "failed to populate container, %s", buf.String())
	}
	log.Println("buffer: ", buf.String())
	log.Println("Populated files for volume: ", volMount.Name)
	return nil
}
