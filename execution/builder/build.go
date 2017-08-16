package build

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/acr-builder/execution/constants"
	"github.com/Azure/acr-builder/execution/domain"
	"github.com/Azure/acr-builder/execution/shell"
	"github.com/sirupsen/logrus"
)

func Run(buildNumber, composeFile, gitURL, dockeruser, dockerpw, registry, gitCloneDir, gitbranch, gitPATokenUser, gitPAToken, gitXToken string, buildEnvs, buildArgs []string, nopublish bool) error {
	envSet := map[string]bool{}
	global := []domain.EnvVar{
		domain.EnvVar{Name: constants.BuildNumberVar, Value: *domain.Abstract(buildNumber)},
		domain.EnvVar{Name: constants.DockerRegistryVar, Value: *domain.Abstract(registry)},
	}
	for _, env := range global {
		envSet[env.Name] = true
	}
	for _, env := range buildEnvs {
		k, v, err := parseAssignment(env)
		if err != nil {
			return fmt.Errorf("Error parsing build environment \"%s\"", err)
		}
		if envSet[k] {
			return fmt.Errorf("Ambiguous environmental variable %s", k)
		}
		global = append(global, domain.EnvVar{Name: k, Value: *domain.Abstract(v)})
		envSet[k] = true
	}

	registryInput := *domain.Abstract(registry)
	dockerAuths := []domain.DockerAuthentication{}
	if (dockeruser == "") != (dockerpw == "") {
		return fmt.Errorf("Please provide both docker user and password or none")
	}
	if dockeruser != "" {
		dockerAuths = append(dockerAuths, domain.DockerAuthentication{
			Registry: registryInput,
			Auth:     domain.NewDockerUsernamePassword(registryInput, dockeruser, dockerpw),
		})
	}

	var gitCred domain.GitCredential
	if gitXToken != "" {
		gitCred = domain.NewXToken(gitXToken)
	} else {
		if (gitPATokenUser == "") != (gitPAToken == "") {
			return fmt.Errorf("Please provide both git user and token or none")
		}

		if gitPATokenUser != "" {
			gitCred = domain.NewGitPersonalAccessToken(gitPATokenUser, gitPAToken)
		}
	}
	var source = &domain.GitSource{
		Address:       *domain.Abstract(gitURL),
		TargetDir:     *domain.Abstract(gitCloneDir),
		InitialBranch: *domain.Abstract(gitbranch),
		Credential:    gitCred,
	}

	build, err := domain.NewDockerComposeBuildTarget(source, gitbranch, composeFile, buildArgs)
	if err != nil {
		return err
	}

	return executeRequest(shell.Instances["sh"],
		&domain.BuildRequest{
			Global:      global,
			DockerAuths: dockerAuths,
			Source:      source,
			Build:       []domain.BuildTarget{*build},
		},
		!nopublish)
}

func executeRequest(sh *shell.Shell, request *domain.BuildRequest, publishOnSuccess bool) error {
	var err error
	var runner domain.Runner
	initialVar := append(
		append(request.Global,
			domain.EnvVar{
				Name:  constants.PublishOnSuccessVar,
				Value: *domain.Abstract(strconv.FormatBool(publishOnSuccess)),
			}),
		request.Source.Export()...)
	runner, err = shell.NewRunner(sh, initialVar)
	if err != nil {
		return fmt.Errorf("Failed to initiate runner, error: %s", err)
	}

	err = request.Source.EnsureSource(runner)
	if err != nil {
		return fmt.Errorf("Failed to ensource source: %s", err)
	}

	err = runIfExists(runner, request.Setup)
	if err != nil {
		return fmt.Errorf("Setup failed: %s", err)
	}

	for _, buildTarget := range request.Build {
		env := buildTarget.Export()
		var buildRunner domain.Runner
		buildRunner, err = runner.AppendContext(env)
		if err != nil {
			return fmt.Errorf("Error initializing build context: %s", err)
		}
		for _, auth := range request.DockerAuths {
			err = auth.Auth.Execute(buildRunner)
			if err != nil {
				return fmt.Errorf("Failed to login: %s", err)
			}
		}
		err = handleBuildAndPublish(
			buildRunner,
			request.DockerAuths,
			buildTarget,
			request.Prebuild,
			request.Postbuild,
			request.Prepublish,
			request.Postpublish,
			publishOnSuccess)
		if err != nil {
			return err
		}
	}

	err = runIfExists(runner, request.WrapUp)
	if err != nil {
		return fmt.Errorf("Wrap up task failed: %s", err)
	}

	return nil
}

func handleBuildAndPublish(
	buildRunner domain.Runner,
	dockerAuths []domain.DockerAuthentication,
	buildTarget domain.BuildTarget,
	prebuild domain.Task,
	postbuild domain.Task,
	prepublish domain.Task,
	postpublish domain.Task,
	publishOnSuccess bool) (err error) {

	err = runIfExists(buildRunner, prebuild)
	if err != nil {
		return fmt.Errorf("Failed to run prebuild task, error: %s", err)
	}

	err = buildTarget.Build.Execute(buildRunner)
	if err != nil {
		return fmt.Errorf("Failed building build task, error: %s", err)
	}

	err = runIfExists(buildRunner, postbuild)
	if err != nil {
		return fmt.Errorf("Failed to run postbuild task, error: %s", err)
	}

	if publishOnSuccess {
		err = runIfExists(buildRunner, prepublish)
		if err != nil {
			return fmt.Errorf("Failed to run prepublish task, error: %s", err)
		}

		err = buildTarget.Publish.Execute(buildRunner)
		if err != nil {
			return fmt.Errorf("Fail to publish image, error: %s", err)
		}

		err = runIfExists(buildRunner, postpublish)
		if err != nil {
			return fmt.Errorf("Failed to run postpublish task, error: %s", err)
		}
	}
	return nil
}

func handleDockerLogin(
	buildRunner domain.Runner,
	dockerAuths []domain.DockerAuthentication,
	targetName string) error {

	targetNamespace := getNamespace(targetName)
	namespaceNotRegistry := !strings.Contains(targetNamespace, ".")
	var authMethod domain.DockerAuthenticationMethod
	for _, login := range dockerAuths {
		registry := buildRunner.Resolve(login.Registry)
		if registry == targetNamespace {
			authMethod = login.Auth
			break
		} else if registry == "" && namespaceNotRegistry {
			authMethod = login.Auth
		}
	}

	// TODO: what about the case where default namespace mapped to github.io??
	if authMethod == nil && !namespaceNotRegistry {
		logrus.Warnf("Namespace %s appears to be a registry but no authentication method is provided.", targetNamespace)
	} else if authMethod != nil {
		if err := authMethod.Execute(buildRunner); err != nil {
			return fmt.Errorf("Error loggin in to %s, error: %s", targetNamespace, err)
		}
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
