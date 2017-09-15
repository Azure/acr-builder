package domain

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/constants"
)

// Vocabulary to be used to build commands
var docker = Abstract("docker")
var login = Abstract("login")
var user = Abstract("-u")
var pw = Abstract("-p")
var build = Abstract("build")
var file = Abstract("-f")
var tag = Abstract("-g")
var buildArgsFlag = Abstract("--build-arg")
var push = Abstract("push")
var pwd = Abstract(".")

type BuildTarget struct {
	Build BuildTask
	Push  PushTask
}

type BuildTask interface {
	Execute(runner Runner) error
}

type PushTask interface {
	Execute(runner Runner) error
}

func (t *BuildTarget) Export() []EnvVar {
	exports := []EnvVar{}
	appendExports(exports, t.Build)
	appendExports(exports, t.Push)
	return exports
}

type DockerAuthentication struct {
	Registry AbstractString
	Auth     DockerAuthenticationMethod
}

type DockerAuthenticationMethod interface {
	Execute(runner Runner) error
}

func NewDockerUsernamePassword(registry AbstractString, username string, password string) *DockerUsernamePassword {
	return &DockerUsernamePassword{
		registry: registry,
		username: *Abstract(username),
		password: *AbstractSensitive(password),
	}
}

type DockerUsernamePassword struct {
	registry AbstractString
	username AbstractString
	password AbstractString
}

func (u *DockerUsernamePassword) Execute(runner Runner) error {
	return runner.ExecuteCmd(*docker, *login, *user, u.username, *pw, u.password, u.registry)
}

type DockerCustomAuthentication struct {
	Task Task
}

func (u *DockerCustomAuthentication) Authenticate(runner Runner, registry AbstractString) error {
	return u.Task.Execute(runner)
}

// NOTE: ensure branch is not null when creating the build task
type DockerBuildTask struct {
	source    SourceDescription
	pushTo    AbstractString
	Branch    AbstractString
	Path      AbstractString
	Context   AbstractString
	BuildArgs []AbstractString
}

func (t *DockerBuildTask) Execute(runner Runner) error {
	var err error
	if t.Branch.value != "" {
		err = t.source.EnsureBranch(runner, t.Branch)
		if err != nil {
			return fmt.Errorf("Error while switching to branch %s", t.Branch.value)
		}
	}
	args := []AbstractString{}
	args[0] = *build
	if t.Path.value != "" {
		args = append(args, *file, t.Path)
	}

	if t.pushTo.value != "" {
		args = append(args, *tag, t.pushTo)
	}

	for _, buildArg := range t.BuildArgs {
		args = append(args, *buildArgsFlag, buildArg)
	}

	if t.Context.value != "" {
		args = append(args, t.Context)
	} else {
		args = append(args, *pwd)
	}

	return runner.ExecuteCmd(*docker, args...)
}

func (t *DockerBuildTask) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			Name:  constants.DockerfilePathVar,
			Value: t.Path,
		},
		EnvVar{
			Name:  constants.DockerBuildContextVar,
			Value: t.Context,
		},
		EnvVar{
			Name:  constants.GitBranchVar,
			Value: t.Branch,
		},
	}
}

type DockerPushTask struct {
	pushTo AbstractString
}

func (t *DockerPushTask) Execute(runner Runner) error {
	return runner.ExecuteCmd(*docker, *push, t.pushTo)
}

func (t *DockerPushTask) Export() []EnvVar {
	return []EnvVar{EnvVar{
		Name:  constants.DockerPushImageVar,
		Value: t.pushTo,
	}}
}

func appendExports(input []EnvVar, obj interface{}) []EnvVar {
	exporter, toExport := obj.(EnvExporter)
	if toExport {
		return append(input, exporter.Export()...)
	}
	return input
}
