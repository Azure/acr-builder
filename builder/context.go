// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/util"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	dependenciesRE = regexp.MustCompile(`(\[{"image.*?\])$`)
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(
	volMounts map[string]string,
	volName string,
	workDir string,
	disableWorkDirOverride bool,
	remove bool,
	detach bool,
	envs []string,
	ports []string,
	expose []string,
	privilaged bool,
	user string,
	network string,
	isolation string,
	cpus string,
	entrypoint string,
	containerName string,
	cmd string) ([]string, error) {
	// Command steps previously ran through a host shell to reuse its field splitting.
	// Parse cmd with the task's OS-independent rules and invoke Docker directly so
	// task data cannot be interpreted as host-shell code. Tasks that need expansion,
	// pipelines, redirection, or chaining must invoke a shell inside the container.
	commandArgs, err := splitCommandLine(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse step command")
	}
	if len(commandArgs) == 0 {
		return nil, errors.New("step command is empty")
	}

	args := []string{"docker", "run"}
	if remove {
		args = append(args, "--rm")
	}
	if detach {
		args = append(args, "--detach")
	}
	for _, port := range ports {
		args = append(args, "-p", port)
	}
	for _, exp := range expose {
		args = append(args, "--expose", exp)
	}
	if privilaged {
		args = append(args, "--privileged")
	}
	if user != "" {
		args = append(args, "--user", user)
	}
	if network != "" {
		args = append(args, "--network", network)
	}
	if isolation != "" {
		args = append(args, "--isolation", isolation)
	}
	if cpus != "" {
		args = append(args, "--cpus", cpus)
	}
	if entrypoint != "" {
		args = append(args, "--entrypoint", entrypoint)
	}
	args = append(args, "--name", containerName)
	args = append(args, "--volume", volName+":"+containerWorkspaceDir)
	args = append(args, "--volume", util.DockerSocketVolumeMapping)
	args = append(args, "--volume", homeVol+":"+homeWorkDir)
	if len(volMounts) > 0 {
		for key, val := range volMounts {
			args = append(args, "--volume", key+":"+val)
		}
	}
	args = append(args, "--env", homeEnv)

	// User environment variables come after any defaults.
	// This allows overriding the HOME environment variable for a step.
	// NB: this has the assumption that the underlying runtime handles the case of duplicated
	// environment variables by only keeping the last specified.
	for _, env := range envs {
		args = append(args, "--env", env)
	}

	if !disableWorkDirOverride {
		args = append(args, "--workdir", normalizeWorkDir(workDir))
	}
	args = append(args, commandArgs...)
	return args, nil
}

// getDockerRunArgsForStep populates the args for running a Docker container for the step.
func (b *Builder) getDockerRunArgsForStep(
	volName string,
	stepWorkDir string,
	step *graph.Step,
	entrypoint string,
	cmd string) ([]string, error) {
	if runtime.GOOS == util.WindowsOS && step.Isolation == "" && !step.IsBuildStep() {
		// Use hyperv isolation for non-build steps.
		// Use default isolation for build step to improve performance. It assumes the docker-cli image is compatible with the host os.
		step.Isolation = "hyperv"
	}

	if runtime.GOOS == util.WindowsOS && step.IsBuildStep() {
		// Limit build command container cpu to 1
		step.CPUS = "1"
	}

	var volMounts = make(map[string]string)
	for _, mount := range step.Mounts {
		volMounts[mount.Name] = mount.MountPath
	}

	return b.getDockerRunArgs(
		volMounts,
		volName,
		stepWorkDir,
		step.DisableWorkingDirectoryOverride,
		!step.Keep,
		step.Detach,
		step.Envs,
		step.Ports,
		step.Expose,
		step.Privileged,
		step.User,
		step.Network,
		step.Isolation,
		step.CPUS,
		entrypoint,
		step.ID,
		cmd,
	)
}

func (b *Builder) scrapeDependencies(
	ctx context.Context,
	volName string,
	stepWorkDir string,
	outputDir string,
	dockerfile string,
	sourceContext string,
	tags []string,
	buildArgs []string,
	target string,
	credentials []*graph.RegistryCredential) ([]*image.Dependencies, error) {
	containerName := fmt.Sprintf("acb_dep_scanner_%s", uuid.New())

	args, censoredArgs, err := getScanArgs(
		containerName,
		volName,
		containerWorkspaceDir,
		stepWorkDir,
		dockerfile,
		outputDir,
		tags,
		buildArgs,
		target,
		sourceContext,
		credentials)

	if err != nil {
		return nil, err
	}

	if b.debug {
		log.Printf("Scan args: %v\n", censoredArgs)
	}

	var buf bytes.Buffer
	err = b.procManager.Run(ctx, args, nil, &buf, &buf, "")
	output := strings.TrimSpace(buf.String())
	if err != nil {
		log.Printf("Output from dependency scanning: %s\n", output)
		return nil, err
	}

	return getImageDependencies(output)
}

func getScanArgs(
	containerName string,
	volName string,
	containerWorkspaceDir string,
	stepWorkDir string,
	dockerfile string,
	outputDir string,
	tags []string,
	buildArgs []string,
	target string,
	sourceContext string,
	credentials []*graph.RegistryCredential) ([]string, []string, error) {
	args := []string{
		"docker",
		"run",
		"--rm",
		"--name", containerName,
		"--volume", volName + ":" + containerWorkspaceDir,
		"--workdir", normalizeWorkDir(stepWorkDir),

		// Mount home
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", homeEnv,

		scannerImageName,
		"scan",
		"-f", dockerfile,
		"--destination", outputDir,
	}

	for _, tag := range tags {
		args = append(args, "-t", tag)
	}

	for _, buildArg := range buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	var censoredArgs = make([]string, len(args))
	copy(censoredArgs, args)

	for _, cred := range credentials {
		serializedCredential, err := cred.String()
		if err != nil {
			return nil, nil, errors.New("credential serialization failed for given registry credential")
		}
		censoredArgs = append(censoredArgs, "--credential", "***")
		args = append(args, "--credential", serializedCredential)
	}

	if len(target) > 0 {
		censoredArgs = append(censoredArgs, "--target", target)
		args = append(args, "--target", target)
	}

	// Positional context must appear last
	censoredArgs = append(censoredArgs, sourceContext)
	args = append(args, sourceContext)
	return args, censoredArgs, nil
}

func getImageDependencies(s string) ([]*image.Dependencies, error) {
	var deps []*image.Dependencies
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		matches := dependenciesRE.FindStringSubmatch(line)
		if len(matches) == 2 {
			err := json.Unmarshal([]byte(matches[1]), &deps)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	return deps, nil
}

// normalizeWorkDir normalizes a working directory.
func normalizeWorkDir(workDir string) string {
	// If the directory is absolute, use it instead of /workspace
	if path.IsAbs(workDir) {
		return path.Clean(workDir)
	}

	return path.Clean(path.Join(containerWorkspaceDir, workDir))
}
