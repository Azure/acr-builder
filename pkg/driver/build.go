package driver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/acr-builder/pkg/grok"
	"github.com/Azure/acr-builder/pkg/workflow"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/constants"
)

// Builder is our main builder
type Builder struct {
	runner build.Runner
}

// NewBuilder creates a builder
func NewBuilder(runner build.Runner) *Builder {
	return &Builder{runner: runner}
}

// Run is the main body of the acr-builder
func (b *Builder) Run(buildNumber,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry,
	workingDir,
	gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
	webArchive string,
	buildEnvs, buildArgs, buildSecretArgs []string, pull, noCache, push bool,
	tags []string) (dependencies []build.ImageDependencies, err error) {

	if dockerRegistry == "" {
		dockerRegistry = os.Getenv(constants.ExportsDockerRegistry)
	}

	var userDefined []build.EnvVar
	userDefined, err = parseUserDefined(buildEnvs)
	if err != nil {
		return
	}

	var request *build.Request
	request, err = b.createBuildRequest(
		dockerfile, dockerImage, dockerContextDir,
		dockerUser, dockerPW, dockerRegistry,
		workingDir,
		gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
		webArchive,
		buildArgs, buildSecretArgs, pull, noCache, push, tags)

	if err != nil {
		return
	}

	buildWorkflow := compileWorkflow(buildNumber, userDefined, request, push)
	err = buildWorkflow.Run(b.runner)
	if err == nil {
		dependencies = buildWorkflow.GetOutputs().ImageDependencies
	}

	return
}

func (b *Builder) createBuildRequest(
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry,
	workingDir,
	gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
	webArchive string,
	buildArgs, buildSecretArgs []string, pull, noCache, push bool, tags []string) (*build.Request, error) {
	if push && dockerRegistry == "" {
		return nil, fmt.Errorf("Docker registry is needed for push, use --%s or environment variable %s to provide its value",
			constants.ArgNameDockerRegistry, constants.ExportsDockerRegistry)
	}

	if push && dockerImage == "" {
		return nil, fmt.Errorf("Docker image is needed for push, use --%s or environment variable %s to provide its value",
			constants.ArgNameDockerImage, constants.ExportsDockerPushImage)
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

	dockerCreds := []build.DockerCredential{}
	if dockerUser != "" {
		cred, err := commands.NewDockerUsernamePassword(registryNoSuffix, dockerUser, dockerPW)
		if err != nil {
			return nil, err
		}
		dockerCreds = append(dockerCreds, cred)
	}

	source, err := getSource(workingDir, gitURL, gitBranch, gitHeadRev, gitXToken, gitPATokenUser, gitPAToken, webArchive)
	if err != nil {
		return nil, err
	}

	target := commands.NewDockerBuild(dockerfile, dockerContextDir, buildArgs, buildSecretArgs, registrySuffixed, dockerImage, pull, noCache, tags)

	return &build.Request{
		DockerRegistry:    registrySuffixed,
		DockerCredentials: dockerCreds,
		Targets: []build.SourceTarget{
			{
				Source: source,
				Builds: []build.Target{target},
			},
		},
	}, nil
}

func parseUserDefined(buildEnvs []string) ([]build.EnvVar, error) {
	userDefined := []build.EnvVar{}
	for _, env := range buildEnvs {
		k, v, err := grok.ParseAssignment(env)
		if err != nil {
			return nil, fmt.Errorf("Error parsing build environment \"%s\"", err)
		}
		envVar, err := build.NewEnvVar(k, v)
		if err != nil {
			return nil, err
		}
		userDefined = append(userDefined, *envVar)
	}
	return userDefined, nil
}

func dependencyTask(target build.Target) workflow.EvaluationTask {
	return func(runner build.Runner, outputContext *workflow.OutputContext) error {
		dependencies, err := target.ScanForDependencies(runner)
		if err != nil {
			return err
		}
		outputContext.ImageDependencies = append(outputContext.ImageDependencies, dependencies...)
		return nil
	}
}

func digestsTask(runner build.Runner, outputContext *workflow.OutputContext) error {
	err := commands.PopulateDigests(runner, outputContext.ImageDependencies)
	return err
}

// a utility type to hold RunningTask, used only for compile method
type runningTaskItem struct {
	context *build.BuilderContext
	task    workflow.RunningTask
}

// a utility type to hold EvaluationTask, used only for compile method
type evaluationTaskItem struct {
	context *build.BuilderContext
	task    workflow.EvaluationTask
}

// compileWorkflow takes a build request and populate it into workflow
func compileWorkflow(buildNumber string,
	userDefined []build.EnvVar, request *build.Request, push bool) *workflow.Workflow {

	// create a workflow with root context with default variables
	w := workflow.NewWorkflow()
	rootContext := build.NewContext(userDefined, []build.EnvVar{
		{Name: constants.ExportsBuildNumber, Value: buildNumber},
		{Name: constants.ExportsBuildTimestamp, Value: time.Now().UTC().Format(constants.TimestampFormat)},
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
		// because pushes need to be run at the end when all builds succeed
		pushItems := []runningTaskItem{}

		// digests tasks will be put into an array and be added at the end
		// when all builds and pushes (if any) succeed
		digestItems := []evaluationTaskItem{}

		// iterate through builds in the source
		for _, build := range sourceTarget.Builds {
			buildContext := sourceContext.Append(build.Export())
			// schedule evaluations of dependencies
			w.ScheduleEvaluation(buildContext, dependencyTask(build))
			// schedule the build tasks
			w.ScheduleRun(buildContext, build.Build)
			// if push is enabled, add them to an array to be added last
			if push {
				pushItems = append(pushItems, runningTaskItem{
					context: buildContext,
					task:    build.Push,
				})
			}

			digestItems = append(digestItems, evaluationTaskItem{
				context: buildContext,
				task:    digestsTask,
			})
		}

		// add the push tasks if there's any
		for _, item := range pushItems {
			w.ScheduleRun(item.context, item.task)
		}

		// add the digest tasks
		for _, item := range digestItems {
			w.ScheduleEvaluation(item.context, item.task)
		}

		// schedule the source return task
		w.ScheduleRun(sourceContext, source.Return)
	}

	return w
}
