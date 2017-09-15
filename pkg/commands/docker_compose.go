package commands

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/gork"
	"github.com/sirupsen/logrus"
)

// This array needs to be an exact copy of docker compose's SUPPORTED_FILENAMES in config.py
var dockerComposeSupportedFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

var dockerCompose = domain.Abstract("docker-compose")
var pull = domain.Abstract("--pull")
var projectDirectory = domain.Abstract("--project-directory")

func NewDockerComposeBuildTarget(source domain.SourceDescription, branch, path, projectDir string, buildArgs []string) (*domain.BuildTarget, error) {
	return &domain.BuildTarget{
		Build: &DockerComposeBuildTask{
			source:           source,
			branch:           *domain.Abstract(branch),
			path:             *domain.Abstract(path),
			projectDirectory: *domain.Abstract(projectDir),
			buildArgs:        domain.AbstractBatch(buildArgs),
		},
		Push: &DockerComposePushTask{
			path: *domain.Abstract(path),
		},
	}, nil
}

type DockerComposeBuildTask struct {
	source           domain.SourceDescription
	branch           domain.AbstractString
	path             domain.AbstractString
	projectDirectory domain.AbstractString
	buildArgs        []domain.AbstractString
}

func (t *DockerComposeBuildTask) Execute(runner domain.Runner) ([]domain.ImageDependencies, error) {
	if !t.branch.IsEmpty() {
		err := t.source.EnsureBranch(runner, t.branch)
		if err != nil {
			return nil, fmt.Errorf("Error while switching to branch %s", t.branch.DisplayValue())
		}
	}

	args := []domain.AbstractString{}
	var targetPath domain.AbstractString
	if !t.path.IsEmpty() {
		targetPath = t.path
	} else {
		var exists bool
		for _, defaultFile := range dockerComposeSupportedFilenames {
			targetPath = *domain.Abstract(defaultFile)
			exists, err := runner.DoesFileExist(targetPath)
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

	dependencies, err := gork.ResolveDockerComposeDependencies(runner, runner.Resolve(t.projectDirectory), runner.Resolve(targetPath))
	if err != nil {
		// Don't fail because we can't figure out dependencies
		logrus.Errorf("Failed to resolve dependencies for docker-compose file %s", targetPath.DisplayValue())
	}

	// TODO: now scan for target path
	args = append(args, *file, targetPath, *build, *pull)

	if !t.projectDirectory.IsEmpty() {
		args = append(args, *projectDirectory, t.projectDirectory)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, *buildArgsFlag, buildArg)
	}

	return dependencies, runner.ExecuteCmd(*dockerCompose, args)
}

func (t *DockerComposeBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		domain.EnvVar{
			Name:  constants.DockerComposeFileVar,
			Value: t.path,
		},
		domain.EnvVar{
			Name:  constants.GitBranchVar,
			Value: t.branch,
		},
	}
}

type DockerComposePushTask struct {
	path domain.AbstractString
}

func (t *DockerComposePushTask) Execute(runner domain.Runner) error {
	args := []domain.AbstractString{}
	if !t.path.IsEmpty() {
		args = append(args, *file, t.path)
	}
	args = append(args, *push)
	return runner.ExecuteCmd(*dockerCompose, args)
}
