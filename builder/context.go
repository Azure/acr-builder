package builder

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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

func (b *Builder) copyContext(ctx context.Context, workingDir string, mountLocation string) error {
	containerName := fmt.Sprintf("rally_context_share_%s", uuid.New())
	cArgs := []string{
		"docker",
		"create",
		"--name", containerName,
		"--volume", b.workspaceDir + ":" + containerWorkspaceDir,

		"--volume", rallyHomeVol + ":" + rallyHomeWorkDir,
		// Set $HOME to the home volume.
		"--env", "HOME=" + rallyHomeVol,
		defaultBlankImage,
	}

	err := b.cmder.Run(ctx, cArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to create container %s to share context", containerName)
	}

	cpArgs := []string{
		"docker",
		"cp",
		workingDir,
		fmt.Sprintf("%s:%s", containerName, normalizeWorkDir(mountLocation)),
	}

	err = b.cmder.Run(ctx, cpArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to copy context to container %s", containerName)
	}

	rmArgs := []string{
		"docker",
		"rm",
		containerName,
	}

	err = b.cmder.Run(ctx, rmArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to clean up shared context container %s", containerName)
	}

	return nil
}

// normalizeWorkDir normalizes a step's working directory.
func normalizeWorkDir(stepWorkDir string) string {
	// If the step's directory is absolute, use it instead of /workspace/...
	if path.IsAbs(stepWorkDir) {
		return path.Clean(stepWorkDir)
	}

	return path.Clean(path.Join("/workspace", stepWorkDir))
}
