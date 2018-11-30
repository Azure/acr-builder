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
	dependenciesRE = regexp.MustCompile(`^(\[{"image.*?\])$`)
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(
	volName string,
	stepWorkDir string,
	step *graph.Step,
	envs []string,
	entrypoint string,
	cmd string) []string {

	var args []string
	var sb strings.Builder
	// Run user commands from a shell instance in order to mirror the shell's field splitting algorithms,
	// so we don't have to write our own argv parser for exec.Command.
	if runtime.GOOS == "windows" {
		args = []string{"powershell.exe", "-Command"}
		if step.Isolation == "" && !step.IsBuildStep() {
			// Use hyperv isolation for non-build steps.
			// Use default isolation for build step to improve performance. It assumes the docker-cli image is compatible with the host os.
			step.Isolation = "hyperv"
		}
	} else {
		args = []string{"/bin/sh", "-c"}
	}

	sb.WriteString("docker run")
	if !step.Keep {
		sb.WriteString(" --rm")
	}
	if step.Detach {
		sb.WriteString(" --detach")
	}
	for _, port := range step.Ports {
		sb.WriteString(" -p " + port)
	}
	for _, exp := range step.Expose {
		sb.WriteString(" --expose " + exp)
	}
	if step.Privileged {
		sb.WriteString(" --privileged")
	}
	if step.User != "" {
		sb.WriteString(" --user " + step.User)
	}
	if step.Network != "" {
		sb.WriteString(" --network " + step.Network)
	}
	if step.Isolation != "" {
		sb.WriteString(" --isolation " + step.Isolation)
	}
	for _, env := range envs {
		sb.WriteString(" --env " + env)
	}
	if entrypoint != "" {
		sb.WriteString(" --entrypoint " + entrypoint)
	}
	sb.WriteString(" --name " + step.ID)
	sb.WriteString(" --volume " + volName + ":" + containerWorkspaceDir)
	sb.WriteString(" --volume " + util.DockerSocketVolumeMapping)
	sb.WriteString(" --volume " + homeVol + ":" + homeWorkDir)
	sb.WriteString(" --env " + homeEnv)
	if !step.DisableWorkingDirectoryOverride {
		sb.WriteString(" --workdir " + normalizeWorkDir(stepWorkDir))
	}
	sb.WriteString(" " + cmd)

	args = append(args, sb.String())
	return args
}

func (b *Builder) scrapeDependencies(ctx context.Context, volName string, stepWorkDir string, outputDir string, dockerfile string, context string, tags []string, buildArgs []string) ([]*image.Dependencies, error) {
	containerName := fmt.Sprintf("acb_dep_scanner_%s", uuid.New())
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
		context,
	}

	for _, tag := range tags {
		args = append(args, "-t", tag)
	}

	for _, buildArg := range buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	if b.debug {
		log.Printf("Scraping args: %v\n", args)
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
