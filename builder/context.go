package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/google/uuid"
)

const (
	defaultScannerImage = "scanner"
)

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(stepID string, stepWorkDir string) []string {
	args := []string{"docker", "run"}

	if rmContainer {
		args = append(args, "--rm")
	}

	args = append(args,
		"--name", fmt.Sprintf("rally_step_%s", stepID),
		"--volume", "/var/run/docker.sock:/var/run/docker.sock",
		"--volume", b.workspaceDir+":"+containerWorkspaceDir,
		"--workdir", normalizeWorkDir(stepWorkDir),

		"--volume", rallyHomeVol+":"+rallyHomeWorkDir,

		// Set $HOME to the home volume.
		"--env", "HOME="+rallyHomeVol,
		"--privileged",
	)
	return args
}

func (b *Builder) scrapeDependencies(ctx context.Context, volName string, outputDir string, dockerfile string, context string, tags []string, buildArgs []string) ([]*models.ImageDependencies, error) {
	containerName := fmt.Sprintf("rally_dep_scanner_%s", uuid.New())
	args := []string{
		"docker",
		"run",
		"--rm",
		"--name", containerName,
		"--volume", volName + ":" + containerWorkspaceDir,
		"--volume", rallyHomeVol + ":" + rallyHomeWorkDir,
		"--env", "HOME=" + rallyHomeVol,
		"--workdir", containerWorkspaceDir,
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

	var buf bytes.Buffer
	err := b.cmder.Run(ctx, args, nil, &buf, &buf, "")
	if err != nil {
		return nil, err
	}

	var deps []*models.ImageDependencies
	bytes := bytes.TrimSpace(buf.Bytes())
	err = json.Unmarshal(bytes, &deps)
	if err != nil {
		return nil, err
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
