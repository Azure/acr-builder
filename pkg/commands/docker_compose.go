package commands

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/grok"
	"github.com/sirupsen/logrus"
)

// This array needs to be an exact copy of docker compose's SUPPORTED_FILENAMES in config.py
var dockerComposeSupportedFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

// NewDockerComposeBuildTarget creates a build target with defined docker-compose file
func NewDockerComposeBuildTarget(source domain.SourceDescription, branch, path, projectDir string, buildArgs []string) *domain.BuildTarget {
	return &domain.BuildTarget{
		Build: &dockerComposeBuildTask{
			source:           source,
			branch:           branch,
			path:             path,
			projectDirectory: projectDir,
			buildArgs:        buildArgs,
		},
		Push: &dockerComposePushTask{
			path: path,
		},
	}
}

type dockerComposeBuildTask struct {
	source           domain.SourceDescription
	branch           string
	path             string
	projectDirectory string
	buildArgs        []string
}

func (t *dockerComposeBuildTask) Execute(runner domain.Runner) ([]domain.ImageDependencies, error) {
	if t.branch != "" {
		err := t.source.EnsureBranch(runner, t.branch)
		if err != nil {
			return nil, fmt.Errorf("Error while switching to branch %s, error: %s", runner.Resolve(t.branch), err)
		}
	}

	args := []string{}
	var targetPath string
	if t.path != "" {
		targetPath = t.path
	} else {
		var exists bool
		for _, defaultFile := range dockerComposeSupportedFilenames {
			targetPath = defaultFile
			var err error
			exists, err = runner.DoesFileExist(targetPath)
			if err != nil {
				logrus.Errorf("Unexpected error while checking for default docker compose file: %s", err)
			}
			if exists {
				break
			}
		}
		if !exists {
			return nil, fmt.Errorf("Failed to find docker compose file in source directory")
		}
	}

	dependencies, err := grok.ResolveDockerComposeDependencies(runner, runner.Resolve(t.projectDirectory), runner.Resolve(targetPath))
	if err != nil {
		// Don't fail because we can't figure out dependencies
		logrus.Errorf("Failed to resolve dependencies for docker-compose file %s", runner.Resolve(targetPath))
	}

	args = append(args, "-f", targetPath, "build", "--pull")

	if t.projectDirectory != "" {
		args = append(args, "--project-directory", t.projectDirectory)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	return dependencies, runner.ExecuteCmd("docker-compose", args)
}

func (t *dockerComposeBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.DockerComposeFileVar,
			Value: t.path,
		},
		{
			Name:  constants.GitBranchVar,
			Value: t.branch,
		},
	}
}

type dockerComposePushTask struct {
	path string
}

func (t *dockerComposePushTask) Execute(runner domain.Runner) error {
	args := []string{}
	if t.path != "" {
		args = append(args, "-f", t.path)
	}
	args = append(args, "push")
	return runner.ExecuteCmd("docker-compose", args)
}
