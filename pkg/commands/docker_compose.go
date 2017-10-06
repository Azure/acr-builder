package commands

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/grok"
)

// DockerComposeSupportedFilenames needs to always be in sync with docker compose's SUPPORTED_FILENAMES in config.py
var DockerComposeSupportedFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

// ErrNoDefaultDockerfile means that no default docker file is found when running FindDefaultDockerComposeFile
var ErrNoDefaultDockerfile = fmt.Errorf("No default docker-compose file found")

// FindDefaultDockerComposeFile try and locate the default docker-compose file in the current working directory
func FindDefaultDockerComposeFile(runner domain.Runner) (string, error) {
	fs := runner.GetFileSystem()
	for _, defaultFile := range DockerComposeSupportedFilenames {
		exists, err := fs.DoesFileExist(defaultFile)
		if err != nil {
			return "", fmt.Errorf("Unexpected error while checking for default docker compose file: %s", err)
		}
		if exists {
			return defaultFile, nil
		}
	}
	return "", ErrNoDefaultDockerfile
}

// NewDockerComposeBuild creates a build target with defined docker-compose file
func NewDockerComposeBuild(path, projectDir string, buildArgs []string) domain.BuildTarget {
	return &dockerComposeBuildTask{
		path:             path,
		projectDirectory: projectDir,
		buildArgs:        buildArgs,
	}
}

type dockerComposeBuildTask struct {
	path             string
	projectDirectory string
	buildArgs        []string
}

func (t *dockerComposeBuildTask) ScanForDependencies(runner domain.Runner) ([]domain.ImageDependencies, error) {
	env := runner.GetContext()
	var targetPath string
	if t.path != "" {
		targetPath = env.Expand(t.path)
	} else {
		var err error
		targetPath, err = FindDefaultDockerComposeFile(runner)
		if err != nil {
			return []domain.ImageDependencies{}, err
		}
	}
	return grok.ResolveDockerComposeDependencies(env, env.Expand(t.projectDirectory), targetPath)
}

func (t *dockerComposeBuildTask) Build(runner domain.Runner) error {
	args := []string{}
	if t.path != "" {
		args = append(args, "-f", t.path)
	}
	args = append(args, "build")

	if t.projectDirectory != "" {
		args = append(args, "--project-directory", t.projectDirectory)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	return runner.ExecuteCmd("docker-compose", args)
}

func (t *dockerComposeBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.ExportsDockerComposeFile,
			Value: t.path,
		},
	}
}

func (t *dockerComposeBuildTask) Push(runner domain.Runner) error {
	args := []string{}
	if t.path != "" {
		args = append(args, "-f", t.path)
	}
	args = append(args, "push")
	return runner.ExecuteCmd("docker-compose", args)
}
