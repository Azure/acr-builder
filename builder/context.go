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
	cmd string) []string {
	var args []string
	var sb strings.Builder
	// Run user commands from a shell instance in order to mirror the shell's field splitting algorithms,
	// so we don't have to write our own argv parser for exec.Command.
	if runtime.GOOS == util.WindowsOS {
		args = []string{"powershell.exe", "-Command"}
	} else {
		args = []string{"/bin/sh", "-c"}
	}

	sb.WriteString("docker run")
	if remove {
		sb.WriteString(" --rm")
	}
	if detach {
		sb.WriteString(" --detach")
	}
	for _, port := range ports {
		sb.WriteString(" -p " + port)
	}
	for _, exp := range expose {
		sb.WriteString(" --expose " + exp)
	}
	if privilaged {
		sb.WriteString(" --privileged")
	}
	if user != "" {
		sb.WriteString(" --user " + user)
	}
	if network != "" {
		sb.WriteString(" --network " + network)
	}
	if isolation != "" {
		sb.WriteString(" --isolation " + isolation)
	}
	if cpus != "" {
		sb.WriteString(" --cpus " + cpus)
	}
	if entrypoint != "" {
		sb.WriteString(" --entrypoint " + entrypoint)
	}
	sb.WriteString(" --name " + containerName)
	sb.WriteString(" --volume " + volName + ":" + containerWorkspaceDir)
	sb.WriteString(" --volume " + util.DockerSocketVolumeMapping)
	sb.WriteString(" --volume " + homeVol + ":" + homeWorkDir)
	if len(volMounts) > 0 {
		for key, val := range volMounts {
			sb.WriteString(" --volume " + key + ":" + val)
		}
	}
	sb.WriteString(" --env " + homeEnv)

	// User environment variables come after any defaults.
	// This allows overriding the HOME environment variable for a step.
	// NB: this has the assumption that the underlying runtime handles the case of duplicated
	// environment variables by only keeping the last specified.
	for _, env := range envs {
		sb.WriteString(" --env " + env)
	}

	if !disableWorkDirOverride {
		sb.WriteString(" --workdir " + normalizeWorkDir(workDir))
	}
	sb.WriteString(" " + cmd)

	args = append(args, sb.String())
	return args
}

// getDockerRunArgsForStep populates the args for running a Docker container for the step.
func (b *Builder) getDockerRunArgsForStep(
	volName string,
	stepWorkDir string,
	step *graph.Step,
	entrypoint string,
	cmd string) []string {
	// Run user commands from a shell instance in order to mirror the shell's field splitting algorithms,
	// so we don't have to write our own argv parser for exec.Command.
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
	target string) ([]*image.Dependencies, error) {
	containerName := fmt.Sprintf("acb_dep_scanner_%s", uuid.New())

	args := getScanArgs(
		containerName,
		volName,
		containerWorkspaceDir,
		stepWorkDir,
		dockerfile,
		outputDir,
		tags,
		buildArgs,
		target,
		sourceContext)

	if b.debug {
		log.Printf("Scan args: %v\n", args)
	}

	var buf bytes.Buffer
	err := b.procManager.Run(ctx, args, nil, &buf, &buf, "")
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
	sourceContext string) []string {
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

	if len(target) > 0 {
		args = append(args, "--target", target)
	}

	// Positional context must appear last
	args = append(args, sourceContext)
	return args
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
