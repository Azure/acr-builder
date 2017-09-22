package commands

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/grok"
)

// Vocabulary to be used to build commands

const defaultDockerfilePath = "Dockerfile"

// NewDockerUsernamePassword creates a authentication object with username and password
func NewDockerUsernamePassword(registry string, username string, password string) domain.DockerAuthenticationMethod {
	return &dockerUsernamePassword{
		registry: registry,
		username: username,
		password: password,
	}
}

type dockerUsernamePassword struct {
	registry string
	username string
	password string
}

func (u *dockerUsernamePassword) Execute(runner domain.Runner) error {
	return runner.ExecuteCmd("docker", []string{"login", "-u", u.username, "-p", u.password, u.registry})
}

// NewDockerBuildTarget creates a build target with specified docker file and build parameters
func NewDockerBuildTarget(source domain.SourceDescription, branch, dockerfile, contextDir string, buildArgs []string, shouldPush bool, registry, imageName string) (*domain.BuildTarget, error) {
	if shouldPush && imageName == "" {
		return nil, fmt.Errorf("When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage)
	}
	var pushTo string
	if registry != "" {
		pushTo = fmt.Sprintf("%s/%s", registry, imageName)
	} else {
		pushTo = ""
	}
	return &domain.BuildTarget{
		Build: &dockerBuildTask{
			source:     source,
			branch:     branch,
			dockerfile: dockerfile,
			contextDir: contextDir,
			buildArgs:  buildArgs,
			pushTo:     pushTo,
		},
		Push: &dockerPushTask{
			pushTo: pushTo,
		},
	}, nil
}

type dockerBuildTask struct {
	source     domain.SourceDescription
	branch     string
	dockerfile string
	contextDir string
	buildArgs  []string
	pushTo     string
}

func (t *dockerBuildTask) Execute(runner domain.Runner) ([]domain.ImageDependencies, error) {
	if t.branch != "" {
		err := t.source.EnsureBranch(runner, t.branch)
		if err != nil {
			return nil, fmt.Errorf("Error while switching to branch %s, error: %s", runner.Resolve(t.branch), err)
		}
	}

	var dockerfile string
	if t.dockerfile == "" {
		dockerfile = defaultDockerfilePath
	} else {
		dockerfile = runner.Resolve(t.dockerfile)
	}

	args := []string{"build"}
	if t.dockerfile != "" {
		args = append(args, "-f", t.dockerfile)
	}

	if t.pushTo != "" {
		args = append(args, "-t", t.pushTo)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	if t.contextDir != "" {
		args = append(args, t.contextDir)
	} else {
		args = append(args, ".")
	}

	var dependencies []domain.ImageDependencies
	runtime, buildtime, err := grok.ResolveDockerfileDependencies(dockerfile)
	if err == nil {
		dependencies = []domain.ImageDependencies{
			domain.ImageDependencies{
				Image:             runner.Resolve(t.pushTo),
				RuntimeDependency: runtime,
				BuildDependencies: buildtime,
			}}
	} else {
		// Don't fail because we can't figure out dependencies
		logrus.Errorf("Failed to resolve dependencies for dockerfile %s", dockerfile)
	}

	return dependencies, runner.ExecuteCmd("docker", args)
}

func (t *dockerBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.DockerfilePathVar,
			Value: t.dockerfile,
		},
		{
			Name:  constants.DockerBuildContextVar,
			Value: t.contextDir,
		},
		{
			Name:  constants.GitBranchVar,
			Value: t.branch,
		},
	}
}

type dockerPushTask struct {
	pushTo string
}

func (t *dockerPushTask) Execute(runner domain.Runner) error {
	if t.pushTo == "" {
		return fmt.Errorf("No push target is defined")
	}
	return runner.ExecuteCmd("docker", []string{"push", t.pushTo})
}

func (t *dockerPushTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.DockerPushImageVar,
			Value: t.pushTo,
		}}
}
