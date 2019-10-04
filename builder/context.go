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
	"os/user"
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

const (
	windowsOS = "windows"
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(
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
	entrypoint string,
	containerName string,
	cmd string,
	adhoc bool,
	debug bool) []string {
	var args []string
	var sb strings.Builder
	// Run user commands from a shell instance in order to mirror the shell's field splitting algorithms,
	// so we don't have to write our own argv parser for exec.Command.
	if runtime.GOOS == windowsOS {
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
	if entrypoint != "" {
		sb.WriteString(" --entrypoint " + entrypoint)
	}
	sb.WriteString(" --name " + containerName)

	if debug {
		// pass in your local dir to container
		volName = getCWD()
		sb.WriteString(" --volume " + volName + ":" + containerWorkspaceDir)
	} else if !adhoc {
		sb.WriteString(" --volume " + volName + ":" + containerWorkspaceDir)
	}
	sb.WriteString(" --volume " + util.DockerSocketVolumeMapping)

	if debug {
		localHomeDir := getHomeDir()
		sb.WriteString(" --volume " + localHomeDir + ":" + homeWorkDir)
	} else {
		sb.WriteString(" --volume " + homeVol + ":" + homeWorkDir)
	}
	sb.WriteString(" --env " + homeEnv)

	// User environment variables come after any defaults.
	// This allows overriding the HOME environment variable for a step.
	// NB: this has the assumption that the underlying runtime handles the case of duplicated
	// environment variables by only keeping the last specified.
	for _, env := range envs {
		sb.WriteString(" --env " + env)
	}

	if debug {
		sb.WriteString(" --workdir " + normalizeWorkDir(containerWorkspaceDir))
	} else if !adhoc && !disableWorkDirOverride {
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
	cmd string,
	adhoc bool) []string {
	// Run user commands from a shell instance in order to mirror the shell's field splitting algorithms,
	// so we don't have to write our own argv parser for exec.Command.
	if runtime.GOOS == windowsOS && step.Isolation == "" && !step.IsBuildStep() {
		// Use hyperv isolation for non-build steps.
		// Use default isolation for build step to improve performance. It assumes the docker-cli image is compatible with the host os.
		step.Isolation = "hyperv"
	}

	return b.getDockerRunArgs(
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
		entrypoint,
		step.ID,
		cmd,
		adhoc,
		b.debug)
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
	adhoc bool,
	debug bool) ([]*image.Dependencies, error) {
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
		sourceContext,
		adhoc,
		debug)

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
	sourceContext string,
	adhoc bool,
	debug bool) []string {
	args := []string{
		"docker",
		"run",
		"--rm",
		"--name", containerName,
	}

	if debug {
		// pass in your local dir to container
		volName = getCWD()
		args = append(args, "--volume", volName+":"+containerWorkspaceDir)
		args = append(args, "--workdir", normalizeWorkDir(stepWorkDir))
	} else if !adhoc {
		args = append(args, "--volume", volName+":"+containerWorkspaceDir)
		args = append(args, "--workdir", normalizeWorkDir(stepWorkDir))
	}

	// Mount home
	if debug {
		localHomeDir := getHomeDir()
		args = append(args, "--volume", localHomeDir+":"+homeWorkDir)
	} else {
		args = append(args, "--volume", homeVol+":"+homeWorkDir)
	}
	args = append(args, "--env", homeEnv)
	args = append(args, scannerImageName)
	args = append(args, "scan")
	args = append(args, "-f", dockerfile)
	args = append(args, "--destination", outputDir)

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

func getCWD() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}
