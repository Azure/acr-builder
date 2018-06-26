package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Azure/acr-builder/util"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/google/uuid"
)

const (
	defaultScannerImage = "scanner"
)

var (
	dependenciesRE = regexp.MustCompile(`^(\[{"image.*?\])$`)
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(volName string, stepID string, stepWorkDir string) []string {
	args := []string{"docker", "run"}

	if rmContainer {
		args = append(args, "--rm")
	}

	// TODO: refactor so that only `build` uses the docker sock.
	args = append(args,
		"--name", fmt.Sprintf("rally_step_%s", stepID),
		"--volume", util.GetDockerSock(),
		"--volume", volName+":"+containerWorkspaceDir,
		"--workdir", normalizeWorkDir(stepWorkDir),
	)
	return args
}

func (b *Builder) scrapeDependencies(ctx context.Context, volName string, stepWorkDir string, outputDir string, dockerfile string, context string, tags []string, buildArgs []string) ([]*models.ImageDependencies, error) {
	containerName := fmt.Sprintf("rally_dep_scanner_%s", uuid.New())
	args := []string{
		"docker",
		"run",
		"--rm",
		"--name", containerName,
		"--volume", volName + ":" + containerWorkspaceDir,
		"--workdir", normalizeWorkDir(stepWorkDir),
		defaultScannerImage,
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
		fmt.Printf("Scraping args: %v\n", args)
	}

	var buf bytes.Buffer
	err := b.cmder.Run(ctx, args, nil, &buf, &buf, "")
	output := strings.TrimSpace(buf.String())
	fmt.Printf("Output from dependency scanning: %v\n", output)
	if err != nil {
		return nil, err
	}

	var deps []*models.ImageDependencies

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := dependenciesRE.FindStringSubmatch(line)

		if len(matches) == 2 {
			err = json.Unmarshal([]byte(matches[1]), &deps)
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
	// If the directory is absolute, use it instead of /workspace/...
	if path.IsAbs(workDir) {
		return path.Clean(workDir)
	}

	return path.Clean(path.Join("/workspace", workDir))
}
