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
	"strings"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/util"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/google/uuid"
)

var (
	dependenciesRE = regexp.MustCompile(`^(\[{"image.*?\])$`)
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(
	volName string,
	stepWorkDir string,
	step *graph.Step) []string {
	args := []string{"docker", "run"}

	if step.Rm {
		args = append(args, "--rm")
	}

	if step.Detach {
		args = append(args, "--detach")
	}

	for _, port := range step.Ports {
		args = append(args, "-p", port)
	}

	if step.Privileged {
		args = append(args, "--privileged")
	}

	if step.User != "" {
		args = append(args, "--user", step.User)
	}

	args = append(args,
		"--name", step.ID,
		"--volume", volName+":"+containerWorkspaceDir,

		// Mount home
		"--volume", util.GetDockerSock(),
		"--volume", homeVol+":"+homeWorkDir,
		"--env", homeEnv,

		"--workdir", normalizeWorkDir(stepWorkDir),
	)
	return args
}

func (b *Builder) scrapeDependencies(ctx context.Context, volName string, stepWorkDir string, outputDir string, dockerfile string, context string, tags []string, buildArgs []string) ([]*models.ImageDependencies, error) {
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
	err := b.taskManager.Run(ctx, args, nil, &buf, &buf, "")
	output := strings.TrimSpace(buf.String())
	if err != nil {
		log.Printf("Output from dependency scanning: %s\n", output)
		return nil, err
	}

	return getImageDependencies(output)
}

func getImageDependencies(s string) ([]*models.ImageDependencies, error) {
	var deps []*models.ImageDependencies
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
