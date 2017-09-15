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

func Run(buildNumber, composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockeruser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitbranch,
	gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource string,
	buildEnvs, buildArgs []string, push bool) ([]domain.ImageDependencies, error) {

	envSet := map[string]bool{}
	global := []domain.EnvVar{
		domain.EnvVar{Name: constants.BuildNumberVar, Value: *domain.Abstract(buildNumber)},
		domain.EnvVar{Name: constants.DockerRegistryVar, Value: *domain.Abstract(dockerRegistry)},
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
		global = append(global, domain.EnvVar{Name: k, Value: *domain.Abstract(v)})
		envSet[k] = true
	}

	registryInput := *domain.Abstract(dockerRegistry)
	dockerAuths := []domain.DockerAuthentication{}
	if (dockeruser == "") != (dockerPW == "") {
		return nil, fmt.Errorf("Please provide both docker user and password or none")
	}
	if dockeruser != "" {
		dockerAuths = append(dockerAuths, domain.DockerAuthentication{
			Registry: registryInput,
			Auth:     commands.NewDockerUsernamePassword(registryInput, dockeruser, dockerPW),
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
		source = &commands.GitSource{
			Address:       *domain.Abstract(gitURL),
			TargetDir:     *domain.Abstract(gitCloneDir),
			HeadRev:       *domain.Abstract(gitHeadRev),
			InitialBranch: *domain.Abstract(gitbranch),
			Credential:    gitCred,
		}
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
		build, err := commands.NewDockerComposeBuildTarget(source, gitbranch, composeFile, composeProjectDir, buildArgs)
		if err != nil {
			return nil, err
		}
		builds = append(builds, *build)
	} else {
		if composeProjectDir != "" {
			return nil, fmt.Errorf("Parameter --%s cannot be used for dockerfile build scenario", constants.ArgNameDockerComposeProjectDir)
		}
		build, err := commands.NewDockerBuildTarget(source, gitbranch, dockerfile, dockerContextDir, buildArgs, push, dockerRegistry, dockerImage)
		if err != nil {
			return nil, err
		}
		builds = append(builds, *build)
	}

	return executeRequest(shell.Instances["sh"],
		&domain.BuildRequest{
			Global:      global,
			DockerAuths: dockerAuths,
			Source:      source,
			Build:       builds,
		},
		push)
}

func executeRequest(sh *shell.Shell, request *domain.BuildRequest, push bool) ([]domain.ImageDependencies, error) {
	var err error
	var runner domain.Runner
	initialVar := append(
		append(request.Global,
			domain.EnvVar{
				Name:  constants.PushOnSuccessVar,
				Value: *domain.Abstract(strconv.FormatBool(push)),
			}),
		request.Source.Export()...)
	runner, err = shell.NewRunner(sh, initialVar)
	if err != nil {
		return nil, fmt.Errorf("Failed to initiate runner, error: %s", err)
	}

	err = request.Source.EnsureSource(runner)
	if err != nil {
		return nil, fmt.Errorf("Failed to ensure source: %s", err)
	}

	err = runIfExists(runner, request.Setup)
	if err != nil {
		return nil, fmt.Errorf("Setup failed: %s", err)
	}

	for _, auth := range request.DockerAuths {
		err = auth.Auth.Execute(runner)
		if err != nil {
			return nil, fmt.Errorf("Failed to login: %s", err)
		}
	}

	allDependencies := []domain.ImageDependencies{}
	for _, buildTarget := range request.Build {
		env := buildTarget.Export()
		var buildRunner domain.Runner
		buildRunner, err = runner.AppendContext(env)
		if err != nil {
			return nil, fmt.Errorf("Error initializing build context: %s", err)
		}
		dep, err := handleBuild(
			buildRunner,
			buildTarget,
			request.Prebuild,
			request.Postbuild)
		if err != nil {
			return nil, err
		}
		allDependencies = append(allDependencies, dep...)
	}

	if push {
		for _, buildTarget := range request.Build {
			env := buildTarget.Export()
			var buildRunner domain.Runner
			buildRunner, err = runner.AppendContext(env)
			if err != nil {
				return nil, fmt.Errorf("Error initializing build context: %s", err)
			}
			err := handlePush(
				buildRunner,
				buildTarget,
				request.Prebuild,
				request.Postbuild)
			if err != nil {
				return nil, err
			}
		}
	}

	err = runIfExists(runner, request.WrapUp)
	if err != nil {
		return nil, fmt.Errorf("Wrap up task failed: %s", err)
	}

	return allDependencies, nil
}

func handleBuild(
	buildRunner domain.Runner,
	buildTarget domain.BuildTarget,
	prebuild domain.Task,
	postbuild domain.Task) ([]domain.ImageDependencies, error) {
	err := runIfExists(buildRunner, prebuild)
	if err != nil {
		return nil, fmt.Errorf("Failed to run prebuild task, error: %s", err)
	}

	dependencies, err := buildTarget.Build.Execute(buildRunner)
	if err != nil {
		return nil, fmt.Errorf("Failed building build task, error: %s", err)
	}

	err = runIfExists(buildRunner, postbuild)
	if err != nil {
		return nil, fmt.Errorf("Failed to run postbuild task, error: %s", err)
	}

	return dependencies, nil
}

func handlePush(
	buildRunner domain.Runner,
	buildTarget domain.BuildTarget,
	prepush domain.Task,
	postpush domain.Task) (err error) {
	err = runIfExists(buildRunner, prepush)
	if err != nil {
		return fmt.Errorf("Failed to run prepush task, error: %s", err)
	}

	err = buildTarget.Push.Execute(buildRunner)
	if err != nil {
		return fmt.Errorf("Fail to push image, error: %s", err)
	}

	err = runIfExists(buildRunner, postpush)
	if err != nil {
		return fmt.Errorf("Failed to run postpush task, error: %s", err)
	}
	return nil
}

func runIfExists(runner domain.Runner, task domain.Task) error {
	if task != nil {
		return task.Execute(runner)
	}
	return nil
}

func getNamespace(image string) string {
	divIndex := strings.LastIndex(image, "/")
	if divIndex == -1 {
		return ""
	}
	return image[0:divIndex]
}

func parseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	return values[0], values[1], nil
}
