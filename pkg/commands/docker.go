package commands

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/gork"
)

// Vocabulary to be used to build commands

const defaultDockerfilePath = "Dockerfile"

var docker = domain.Abstract("docker")
var login = domain.Abstract("login")
var user = domain.Abstract("-u")
var pw = domain.Abstract("-p")
var build = domain.Abstract("build")
var file = domain.Abstract("-f")
var tag = domain.Abstract("-t")
var buildArgsFlag = domain.Abstract("--build-arg")
var push = domain.Abstract("push")
var pwd = domain.Abstract(".")

func NewDockerUsernamePassword(registry domain.AbstractString, username string, password string) *DockerUsernamePassword {
	return &DockerUsernamePassword{
		registry: registry,
		username: *domain.Abstract(username),
		password: *domain.AbstractSensitive(password),
	}
}

type DockerUsernamePassword struct {
	registry domain.AbstractString
	username domain.AbstractString
	password domain.AbstractString
}

func (u *DockerUsernamePassword) Execute(runner domain.Runner) error {
	return runner.ExecuteCmd(*docker, []domain.AbstractString{*login, *user, u.username, *pw, u.password, u.registry})
}

type DockerCustomAuthentication struct {
	Task domain.Task
}

func (u *DockerCustomAuthentication) Authenticate(runner domain.Runner, registry domain.AbstractString) error {
	return u.Task.Execute(runner)
}

func NewDockerBuildTarget(source domain.SourceDescription, branch, dockerfile, contextDir string, buildArgs []string, shouldPush bool, registry, imageName string) (*domain.BuildTarget, error) {
	if shouldPush && imageName == "" {
		return nil, fmt.Errorf("When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage)
	}
	pushTo := fmt.Sprintf("%s/%s", registry, imageName)
	return &domain.BuildTarget{
		Build: &DockerBuildTask{
			source:     source,
			branch:     *domain.Abstract(branch),
			dockerfile: *domain.Abstract(dockerfile),
			contextDir: *domain.Abstract(contextDir),
			buildArgs:  domain.AbstractBatch(buildArgs),
			pushTo:     *domain.Abstract(pushTo),
		},
		Push: &DockerPushTask{
			pushTo: *domain.Abstract(pushTo),
		},
	}, nil
}

type DockerBuildTask struct {
	source     domain.SourceDescription
	branch     domain.AbstractString
	dockerfile domain.AbstractString
	contextDir domain.AbstractString
	buildArgs  []domain.AbstractString
	pushTo     domain.AbstractString
}

func (t *DockerBuildTask) Execute(runner domain.Runner) ([]domain.ImageDependencies, error) {
	if !t.branch.IsEmpty() {
		err := t.source.EnsureBranch(runner, t.branch)
		if err != nil {
			return nil, fmt.Errorf("Error while switching to branch %s", t.branch.DisplayValue())
		}
	}

	var dockerfile string
	if t.dockerfile.IsEmpty() {
		dockerfile = defaultDockerfilePath
	} else {
		dockerfile = runner.Resolve(t.dockerfile)
	}

	args := []domain.AbstractString{*build}
	if !t.dockerfile.IsEmpty() {
		args = append(args, *file, t.dockerfile)
	}

	if !t.pushTo.IsEmpty() {
		args = append(args, *tag, t.pushTo)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, *buildArgsFlag, buildArg)
	}

	if !t.contextDir.IsEmpty() {
		args = append(args, t.contextDir)
	} else {
		args = append(args, *pwd)
	}

	runtime, buildtime, err := gork.ResolveDockerfileDependencies(dockerfile)
	if err != nil {
		// Don't fail because we can't figure out dependencies
		logrus.Errorf("Failed to resolve dependencies for dockerfile %s", dockerfile)
	}

	return []domain.ImageDependencies{
			domain.ImageDependencies{
				Image:             runner.Resolve(t.pushTo),
				RuntimeDependency: runtime,
				BuildDependencies: buildtime,
			}},
		runner.ExecuteCmd(*docker, args)
}

func (t *DockerBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		domain.EnvVar{
			Name:  constants.DockerfilePathVar,
			Value: t.dockerfile,
		},
		domain.EnvVar{
			Name:  constants.DockerBuildContextVar,
			Value: t.contextDir,
		},
		domain.EnvVar{
			Name:  constants.GitBranchVar,
			Value: t.branch,
		},
	}
}

type DockerPushTask struct {
	pushTo domain.AbstractString
}

func (t *DockerPushTask) Execute(runner domain.Runner) error {
	return runner.ExecuteCmd(*docker, []domain.AbstractString{*push, t.pushTo})
}

func (t *DockerPushTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		domain.EnvVar{
			Name:  constants.DockerPushImageVar,
			Value: t.pushTo,
		}}
}
