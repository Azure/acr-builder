package build

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/acr-builder/pkg/workflow"

	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
)

const buildTimestampFormat = time.RFC3339

// Builder is our main builder
type Builder struct {
	runner domain.Runner
}

// NewBuilder creates a builder
func NewBuilder(runner domain.Runner) *Builder {
	return &Builder{runner: runner}
}

// Run is the main body of the acr-builder
func (b *Builder) Run(buildNumber, composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitBranch,
	gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource string,
	buildEnvs, buildArgs []string, push bool) ([]domain.ImageDependencies, error) {

	if dockerRegistry == "" {
		dockerRegistry = os.Getenv(constants.ExportsDockerRegistry)
	}

	userDefined, err := parseUserDefined(buildEnvs)
	if err != nil {
		return nil, err
	}

	request, err := b.createBuildRequest(composeFile, composeProjectDir,
		dockerfile, dockerImage, dockerContextDir,
		dockerUser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitBranch,
		gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource,
		buildArgs, push)

	if err != nil {
		return nil, err
	}

	buildWorkflow := compileWorkflow(buildNumber, userDefined, request, push)
	err = buildWorkflow.Run(b.runner)
	return buildWorkflow.GetOutputs().ImageDependencies, err
}

func (b *Builder) createBuildRequest(composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitBranch,
	gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource string,
	buildArgs []string, push bool) (*domain.BuildRequest, error) {
	if push && dockerRegistry == "" {
		return nil, fmt.Errorf("Docker registry is needed for push, use --%s or environment variable %s to provide its value",
			constants.ArgNameDockerRegistry, constants.ExportsDockerRegistry)
	}

	var registrySuffixed, registryNoSuffix string
	if dockerRegistry != "" {
		if strings.HasSuffix(dockerRegistry, "/") {
			registrySuffixed = dockerRegistry
			registryNoSuffix = dockerRegistry[:len(dockerRegistry)-1]
		} else {
			registrySuffixed = dockerRegistry + "/"
			registryNoSuffix = dockerRegistry
		}
	}

	dockerCreds := []domain.DockerCredential{}
	if dockerUser != "" {
		cred, err := commands.NewDockerUsernamePassword(registryNoSuffix, dockerUser, dockerPW)
		if err != nil {
			return nil, err
		}
		dockerCreds = append(dockerCreds, cred)
	}

	var source domain.SourceTarget
	if gitURL != "" {
		var gitCred commands.GitCredential
		if gitXToken != "" {
			gitCred = commands.NewGitXToken(gitXToken)
		} else {
			if gitPATokenUser != "" {
				var err error
				gitCred, err = commands.NewGitPersonalAccessToken(gitPATokenUser, gitPAToken)
				if err != nil {
					return nil, err
				}
			}
		}
		source.Source = commands.NewGitSource(gitURL, gitBranch, gitHeadRev, gitCloneDir, gitCred)
	} else {
		if gitXToken != "" || gitPATokenUser != "" || gitPAToken != "" {
			return nil, fmt.Errorf("Git credentials are given but --%s was not", constants.ArgNameGitURL)
		}
		source.Source = commands.NewLocalSource(localSource)
	}

	// The following code block tries to determine which kind of build task to include
	// Note: consider the following case
	// (composeFile == "" && dockerImage == "" && dockerfile == "" && push == false)
	// The only way for us to determine whether docker or docker-compose needs to be used
	// is the scan in the source code. Existence of the source cannot be assume exist until
	// the workflow checks out the source
	// Currently the code will actually defaults to docker-compose task and try to proceed
	var build domain.BuildTarget
	if dockerImage != "" || dockerfile != "" {
		if dockerfile == "" {
			logrus.Infof("Docker image is defined, dockerfile will be used for building")
		}
		if composeProjectDir != "" {
			return nil, fmt.Errorf("Parameter --%s cannot be used for dockerfile build scenario", constants.ArgNameDockerComposeProjectDir)
		}
		build = commands.NewDockerBuild(dockerfile, dockerContextDir, buildArgs, registrySuffixed, dockerImage)
	}

	// Use docker-compose as default
	if build == nil {
		if composeFile == "" {
			logrus.Infof("No dockerfile is provided as parameter, using docker-compose as default")
		}
		// sure, dockerfile and dockerImage shouldn't be empty here but it's just here for correctness
		if dockerfile != "" || dockerImage != "" || dockerContextDir != "" {
			return nil, fmt.Errorf("Parameters --%s, --%s, %s cannot be used in docker-compose scenario",
				constants.ArgNameDockerfile, constants.ArgNameDockerImage, constants.ArgNameDockerContextDir)
		}
		build = commands.NewDockerComposeBuild(composeFile, composeProjectDir, buildArgs)
	}
	source.Builds = append(source.Builds, build)

	return &domain.BuildRequest{
		DockerRegistry:    registrySuffixed,
		DockerCredentials: dockerCreds,
		Targets:           []domain.SourceTarget{source},
	}, nil
}

func parseUserDefined(buildEnvs []string) ([]domain.EnvVar, error) {
	userDefined := []domain.EnvVar{}
	for _, env := range buildEnvs {
		k, v, err := parseAssignment(env)
		if err != nil {
			return nil, fmt.Errorf("Error parsing build environment \"%s\"", err)
		}
		envVar, err := domain.NewEnvVar(k, v)
		if err != nil {
			return nil, err
		}
		userDefined = append(userDefined, *envVar)
	}
	return userDefined, nil
}

func parseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	return values[0], values[1], nil
}

func dependencyTask(build domain.BuildTarget) workflow.EvaluationTask {
	return func(runner domain.Runner, outputContext *workflow.OutputContext) error {
		dependencies, err := build.ScanForDependencies(runner)
		if err != nil {
			return err
		}
		outputContext.ImageDependencies = append(outputContext.ImageDependencies, dependencies...)
		return nil
	}
}

// a utility type used only for compile method
type pushItem struct {
	context *domain.BuilderContext
	push    workflow.RunningTask
}

// compileWorkflow takes a build request and populate it into workflow
func compileWorkflow(buildNumber string,
	userDefined []domain.EnvVar, request *domain.BuildRequest, push bool) *workflow.Workflow {

	// create a workflow with root context with default variables
	w := workflow.NewWorkflow()
	rootContext := domain.NewContext(userDefined, []domain.EnvVar{
		{Name: constants.ExportsBuildNumber, Value: buildNumber},
		{Name: constants.ExportsBuildTimestamp, Value: time.Now().UTC().Format(buildTimestampFormat)},
		{Name: constants.ExportsDockerRegistry, Value: request.DockerRegistry},
		{Name: constants.ExportsPushOnSuccess, Value: strconv.FormatBool(push)}})

	// schedule docker authentications
	for _, auth := range request.DockerCredentials {
		w.ScheduleRun(rootContext, auth.Authenticate)
	}

	// iterate through the source targets
	for _, sourceTarget := range request.Targets {
		source := sourceTarget.Source
		sourceContext := rootContext.Append(source.Export())
		// schedule obtaining the source
		w.ScheduleRun(sourceContext, source.Obtain)

		// push tasks (if any) will be put into an array and be added later
		// because pushes need to be run at the end when all builds succeeds
		pushItems := []pushItem{}

		// iterate through builds in the source
		for _, build := range sourceTarget.Builds {
			buildContext := sourceContext.Append(build.Export())
			// schedule evaluations of dependencies
			w.ScheduleEvaluation(buildContext, dependencyTask(build))
			// schedule the build tasks
			w.ScheduleRun(buildContext, build.Build)
			// if push is enabled, add them to an array to be added last
			if push {
				pushItems = append(pushItems, pushItem{
					context: buildContext,
					push:    build.Push,
				})
			}
		}

		// add the push tasks if there's any
		for _, item := range pushItems {
			w.ScheduleRun(item.context, item.push)
		}

		// schedule the source return task
		w.ScheduleRun(sourceContext, source.Return)
	}

	return w
}
