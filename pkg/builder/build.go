package build

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/shell"
)

// Run is the main body of the acr-builder
func Run(buildNumber, composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitBranch,
	gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource string,
	buildEnvs, buildArgs []string, push bool) ([]domain.ImageDependencies, error) {

	envSet := map[string]bool{}
	global := []domain.EnvVar{
		{Name: constants.BuildNumberVar, Value: buildNumber},
		{Name: constants.DockerRegistryVar, Value: dockerRegistry},
	}
	for _, env := range global {
		envSet[env.Name] = true
	}
	for _, env := range buildEnvs {
		k, v, err := parseAssignment(env)
		if err != nil {
			return nil, fmt.Errorf("Error parsing build environment \"%s\"", err)
		}
		if envSet[k] {
			return nil, fmt.Errorf("Ambiguous environmental variable %s", k)
		}
		global = append(global, domain.EnvVar{Name: k, Value: v})
		envSet[k] = true
	}

	dockerAuths := []domain.DockerAuthentication{}
	if (dockerUser == "") != (dockerPW == "") {
		return nil, fmt.Errorf("Please provide both docker user and password or none")
	}
	if dockerUser != "" {
		dockerAuths = append(dockerAuths, domain.DockerAuthentication{
			Registry: dockerRegistry,
			Auth:     commands.NewDockerUsernamePassword(dockerRegistry, dockerUser, dockerPW),
		})
	}

	var source domain.SourceDescription
	if gitURL != "" {
		var gitCred commands.GitCredential
		if gitXToken != "" {
			gitCred = commands.NewXToken(gitXToken)
		} else {
			if (gitPATokenUser == "") != (gitPAToken == "") {
				return nil, fmt.Errorf("Please provide both git user and token or none")
			}
			if gitPATokenUser != "" {
				gitCred = commands.NewGitPersonalAccessToken(gitPATokenUser, gitPAToken)
			}
		}
		source = commands.NewGitSource(gitURL, gitBranch, gitHeadRev, gitCloneDir, gitCred)
	} else {
		var err error
		source, err = domain.NewLocalSource(localSource)
		if err != nil {
			return nil, err
		}
	}

	builds := []domain.BuildTarget{}
	if composeFile != "" {
		if dockerfile != "" || dockerImage != "" || dockerContextDir != "" {
			return nil, fmt.Errorf("Parameters --%s, --%s, %s cannot be used with %s",
				constants.ArgNameDockerfile, constants.ArgNameDockerImage,
				constants.ArgNameDockerContextDir, constants.ArgNameDockerComposeFile)
		}
		build := commands.NewDockerComposeBuildTarget(source, gitBranch, composeFile, composeProjectDir, buildArgs)
		builds = append(builds, *build)
	} else {
		if composeProjectDir != "" {
			return nil, fmt.Errorf("Parameter --%s cannot be used for dockerfile build scenario", constants.ArgNameDockerComposeProjectDir)
		}
		build, err := commands.NewDockerBuildTarget(source, gitBranch, dockerfile, dockerContextDir, buildArgs, push, dockerRegistry, dockerImage)
		if err != nil {
			return nil, err
		}
		builds = append(builds, *build)
	}

	return executeRequest(
		&domain.BuildRequest{
			Global:      global,
			DockerAuths: dockerAuths,
			Source:      source,
			Build:       builds,
		},
		push)
}

func executeRequest(request *domain.BuildRequest, push bool) ([]domain.ImageDependencies, error) {
	var err error
	var runner domain.Runner
	runner = shell.NewRunner(request.Global, append(
		request.Source.Export(),
		domain.EnvVar{
			Name:  constants.PushOnSuccessVar,
			Value: strconv.FormatBool(push),
		},
	))

	err = request.Source.EnsureSource(runner)
	if err != nil {
		return nil, fmt.Errorf("Failed to ensure source: %s", err)
	}

	for _, auth := range request.DockerAuths {
		err = auth.Auth.Execute(runner)
		if err != nil {
			return nil, fmt.Errorf("Failed to login: %s", err)
		}
	}

	allDependencies := []domain.ImageDependencies{}
	for _, buildTarget := range request.Build {
		buildRunner := runner.AppendContext(buildTarget.Export())
		dep, err := handleBuild(
			buildRunner,
			buildTarget)
		if err != nil {
			return nil, err
		}
		allDependencies = append(allDependencies, dep...)
	}

	if push {
		for _, buildTarget := range request.Build {
			buildRunner := runner.AppendContext(buildTarget.Export())
			err := handlePush(
				buildRunner,
				buildTarget)
			if err != nil {
				return nil, err
			}
		}
	}
	return allDependencies, nil
}

func handleBuild(
	buildRunner domain.Runner,
	buildTarget domain.BuildTarget) ([]domain.ImageDependencies, error) {

	dependencies, err := buildTarget.Build.Execute(buildRunner)
	if err != nil {
		return nil, fmt.Errorf("Failed building build task, error: %s", err)
	}
	return dependencies, nil
}

func handlePush(
	buildRunner domain.Runner,
	buildTarget domain.BuildTarget) (err error) {

	err = buildTarget.Push.Execute(buildRunner)
	if err != nil {
		return fmt.Errorf("Fail to push image, error: %s", err)
	}
	return nil
}

func parseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	return values[0], values[1], nil
}
