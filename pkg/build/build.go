package build

import (
	"fmt"
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

	buildWorkflow := compile(buildNumber, dockerRegistry, userDefined, request, push)
	err = buildWorkflow.Run(b.runner)
	return buildWorkflow.GetOutputs().ImageDependencies, err
}

func (b *Builder) createBuildRequest(composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry, gitURL, gitCloneDir, gitBranch,
	gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource string,
	buildArgs []string, push bool) (*domain.BuildRequest, error) {

	dockerCreds := []domain.DockerCredential{}
	if dockerUser != "" {
		cred, err := commands.NewDockerUsernamePassword(dockerRegistry, dockerUser, dockerPW)
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
		var err error
		source.Source, err = commands.NewGitSource(gitURL, gitBranch, gitHeadRev, gitCloneDir, gitCred)
		if err != nil {
			return nil, err
		}
	} else {
		if gitXToken != "" || gitPATokenUser != "" || gitPAToken != "" {
			return nil, fmt.Errorf("Git credentials are given but --%s was not", constants.ArgNameGitURL)
		}
		source.Source = commands.NewLocalSource(localSource)
	}

	var build domain.BuildTarget
	if composeFile == "" {
		// composeFile is not specified, maybe we should try dockerfile... unless we found the default docker-compose.y(a)ml
		defaultComposeFileFound := false
		if dockerfile == "" {
			// ISSUES: without source being obtained, we are not in the correct directory for scanning default compose file
			composeFile, err := commands.FindDefaultDockerComposeFile(b.runner)
			if err != nil && err != commands.ErrNoDefaultDockerfile {
				// Note: do we really exit here? What does it mean when there's an error checking docker-compose file?
				return nil, err
			}
			defaultComposeFileFound = (composeFile != "")
		}
		if !defaultComposeFileFound {
			if composeProjectDir != "" {
				return nil, fmt.Errorf("Parameter --%s cannot be used for dockerfile build scenario", constants.ArgNameDockerComposeProjectDir)
			}
			var err error
			build, err = commands.NewDockerBuild(dockerfile, dockerContextDir, buildArgs, push, dockerRegistry, dockerImage)
			if err != nil {
				return nil, err
			}
		}
	}

	if build == nil {
		if composeFile == "" {
			logrus.Debugf("No dockerfile is provided, using docker-compose as default")
		}
		if dockerfile != "" || dockerImage != "" || dockerContextDir != "" {
			return nil, fmt.Errorf("Parameters --%s, --%s, %s cannot be used in docker-compose scenario",
				constants.ArgNameDockerfile, constants.ArgNameDockerImage, constants.ArgNameDockerContextDir)
		}
		build = commands.NewDockerComposeBuild(composeFile, composeProjectDir, buildArgs)
	}
	source.Builds = append(source.Builds, build)

	return &domain.BuildRequest{
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
			// build continues if dependency scan fails
			logrus.Errorf("Failed to find dependency for build task, error: %s", err)
		} else {
			outputContext.ImageDependencies = append(outputContext.ImageDependencies, dependencies...)
		}
		return nil
	}
}

// a utility type used only for compile method
type pushItem struct {
	context *domain.BuilderContext
	push    workflow.RunningTask
}

// compile takes a build request and populate it into workflow
func compile(buildNumber, dockerRegistry string,
	userDefined []domain.EnvVar, request *domain.BuildRequest, push bool) *workflow.Workflow {

	w := workflow.NewWorkflow()
	rootContext := domain.NewContext(userDefined, []domain.EnvVar{
		{Name: constants.ExportsBuildNumber, Value: buildNumber},
		{Name: constants.ExportsBuildTimestamp, Value: time.Now().UTC().Format(buildTimestampFormat)},
		{Name: constants.ExportsDockerRegistry, Value: dockerRegistry},
		{Name: constants.ExportsPushOnSuccess, Value: strconv.FormatBool(push)}})

	for _, auth := range request.DockerCredentials {
		w.ScheduleRun(rootContext, auth.Authenticate)
	}

	pushItems := []pushItem{}
	for _, sourceTarget := range request.Targets {
		source := sourceTarget.Source
		sourceContext := rootContext.Append(source.Export())
		w.ScheduleRun(sourceContext, source.Obtain)
		for _, build := range sourceTarget.Builds {
			buildContext := sourceContext.Append(build.Export())
			w.ScheduleEvaluation(buildContext, dependencyTask(build))
			w.ScheduleRun(buildContext, build.Build)
			// if push is enabled, add them last
			if push {
				pushItems = append(pushItems, pushItem{
					context: buildContext,
					push:    build.Push,
				})
			}
		}
		w.ScheduleRun(sourceContext, source.Return)
	}

	for _, item := range pushItems {
		w.ScheduleRun(item.context, item.push)
	}

	return w
}
