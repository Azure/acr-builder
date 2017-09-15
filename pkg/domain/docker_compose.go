package domain

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/sirupsen/logrus"
)

// This array needs to be an exact copy of docker compose's SUPPORTED_FILENAMES in config.py
var dockerComposeSupportedFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

var dockerCompose = Abstract("docker-compose")

func NewDockerComposeBuildTarget(source SourceDescription, branch string, path string, buildArgsStr []string) (*BuildTarget, error) {
	buildArgs := make([]AbstractString, len(buildArgsStr))
	for i, v := range buildArgsStr {
		buildArgs[i] = *Abstract(v)
	}

	return &BuildTarget{
		Build: &DockerComposeBuildTask{
			source:    source,
			Branch:    *Abstract(branch),
			Path:      *Abstract(path),
			BuildArgs: buildArgs,
		},
		Push: &DockerComposePushTask{
			path: *Abstract(path),
		},
	}, nil
}

type DockerComposeBuildTask struct {
	source    SourceDescription
	Branch    AbstractString
	Path      AbstractString
	BuildArgs []AbstractString
}

func (t *DockerComposeBuildTask) Execute(runner Runner) error {
	var err error
	if t.Branch.value != "" {
		err = t.source.EnsureBranch(runner, t.Branch)
		if err != nil {
			return fmt.Errorf("Error while switching to branch %s", t.Branch.value)
		}
	}
	args := []AbstractString{}
	var targetPath AbstractString
	if t.Path.value != "" {
		targetPath = t.Path
	} else {
		var exists bool
		for _, defaultFile := range dockerComposeSupportedFilenames {
			targetPath = *Abstract(defaultFile)
			exists, err = runner.DoesFileExist(targetPath)
			if err != nil {
				logrus.Errorf("Unexpected error while checking for default docker compose file: %s", err)
			}
			if exists {
				break
			}
		}
		if !exists {
			return fmt.Errorf("Failed to find docker compose file in source directory")
		}
	}
	// TODO: now scan for target path
	args = append(args, *file, targetPath)
	args = append(args, *build)

	for _, buildArg := range t.BuildArgs {
		args = append(args, *buildArgsFlag, buildArg)
	}

	return runner.ExecuteCmd(*dockerCompose, args...)
}

func (t *DockerComposeBuildTask) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			Name:  constants.DockerComposeFileVar,
			Value: t.Path,
		},
		EnvVar{
			Name:  constants.GitBranchVar,
			Value: t.Branch,
		},
	}
}

type DockerComposePushTask struct {
	path AbstractString
}

func (t *DockerComposePushTask) Execute(runner Runner) error {
	args := []AbstractString{}
	if t.path.value != "" {
		args = append(args, *file, t.path)
	}
	args = append(args, *push)
	return runner.ExecuteCmd(*dockerCompose, args...)
}
