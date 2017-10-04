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
func NewDockerUsernamePassword(registry string, username string, password string) (domain.DockerCredential, error) {
	if (username == "") != (password == "") {
		return nil, fmt.Errorf("Please provide both --%s and --%s or neither", constants.ArgNameDockerUser, constants.ArgNameDockerPW)
	}
	return &dockerUsernamePassword{
		registry: registry,
		username: username,
		password: password,
	}, nil
}

type dockerUsernamePassword struct {
	registry string
	username string
	password string
}

func (u *dockerUsernamePassword) Authenticate(runner domain.Runner) error {
	return runner.ExecuteCmdWithObfuscation(func(args []string) {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-p" {
				args[i+1] = constants.ObfuscationString
				return
			}
		}
		logrus.Errorf("No password found, obfuscation not performed")
	}, "docker", []string{"login", "-u", u.username, "-p", u.password, u.registry})
}

// NewDockerBuild creates a build target with specified docker file and build parameters
func NewDockerBuild(dockerfile, contextDir string,
	buildArgs []string, shouldPush bool, registry, imageName string) (domain.BuildTarget, error) {

	if shouldPush && imageName == "" {
		return nil, fmt.Errorf("When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage)
	}
	var pushTo string
	if registry != "" {
		pushTo = fmt.Sprintf("%s/%s", registry, imageName)
	} else {
		pushTo = ""
	}
	return &dockerBuildTask{
		dockerfile: dockerfile,
		contextDir: contextDir,
		buildArgs:  buildArgs,
		pushTo:     pushTo,
	}, nil
}

type dockerBuildTask struct {
	dockerfile string
	contextDir string
	buildArgs  []string
	pushTo     string
}

func (t *dockerBuildTask) ScanForDependencies(runner domain.Runner) ([]domain.ImageDependencies, error) {
	env := runner.GetContext()
	var dockerfile string
	if t.dockerfile == "" {
		dockerfile = defaultDockerfilePath
	} else {
		dockerfile = env.Expand(t.dockerfile)
	}

	var dependencies []domain.ImageDependencies
	runtime, buildtime, err := grok.ResolveDockerfileDependencies(dockerfile)
	if err == nil {
		dependencies = []domain.ImageDependencies{
			domain.ImageDependencies{
				Image:             env.Expand(t.pushTo),
				RuntimeDependency: runtime,
				BuildDependencies: buildtime,
			}}
	}
	return dependencies, err
}

func (t *dockerBuildTask) Build(runner domain.Runner) error {
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
	return runner.ExecuteCmd("docker", args)
}

func (t *dockerBuildTask) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.ExportsDockerfilePath,
			Value: t.dockerfile,
		},
		{
			Name:  constants.ExportsDockerBuildContext,
			Value: t.contextDir,
		},
		{
			Name:  constants.ExportsDockerPushImage,
			Value: t.pushTo,
		},
	}
}

func (t *dockerBuildTask) Push(runner domain.Runner) error {
	if t.pushTo == "" {
		return fmt.Errorf("No push target is defined")
	}
	return runner.ExecuteCmd("docker", []string{"push", t.pushTo})
}
