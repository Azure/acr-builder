package driver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

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
func (b *Builder) Run(buildNumber, composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry,
	workingDir,
	gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
	webArchive string,
	buildEnvs, buildArgs, buildSecretArgs []string, push bool,
) (dependencies []build.ImageDependencies, duration time.Duration, err error) {
	startTime := time.Now()
	defer func() {
		duration = time.Since(startTime)
	}()

	if dockerRegistry == "" {
		dockerRegistry = os.Getenv(constants.ExportsDockerRegistry)
	}

	var userDefined []build.EnvVar
	userDefined, err = parseUserDefined(buildEnvs)
	if err != nil {
		return
	}

	var request *build.Request
	request, err = b.createBuildRequest(composeFile, composeProjectDir,
		dockerfile, dockerImage, dockerContextDir,
		dockerUser, dockerPW, dockerRegistry,
		workingDir,
		gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
		webArchive,
		buildArgs, buildSecretArgs, push)

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

func (b *Builder) createBuildRequest(composeFile, composeProjectDir,
	dockerfile, dockerImage, dockerContextDir,
	dockerUser, dockerPW, dockerRegistry,
	workingDir,
	gitURL, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
	webArchive string,
	buildArgs, buildSecretArgs []string, push bool) (*build.Request, error) {
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

	// The following code block tries to determine which kind of build task to include
	// Note: consider the following case
	// (composeFile == "" && dockerImage == "" && dockerfile == "" && push == false)
	// The only way for us to determine whether docker or docker-compose needs to be used
	// is the scan in the source code. Existence of the source cannot be assume exist until
	// the workflow checks out the source
	// Currently the code will actually defaults to docker-compose task and try to proceed
	var target build.Target
	if dockerImage != "" || dockerfile != "" {
		if dockerfile == "" {
			logrus.Debugf("Docker image is defined, dockerfile will be used for building")
		}
		if len(dockerImage) <= 0 {
			return nil, fmt.Errorf("Image name not specified for docker file '%s'", dockerfile)
		}
		if composeProjectDir != "" {
			return nil, fmt.Errorf("Parameter --%s cannot be used for dockerfile build scenario", constants.ArgNameDockerComposeProjectDir)
		}
		target = commands.NewDockerBuild(dockerfile, dockerContextDir, buildArgs, buildSecretArgs, registrySuffixed, dockerImage)
	}

	// Use docker-compose as default
	if target == nil {
		if composeFile == "" {
			logrus.Debugf("No dockerfile is provided as parameter, using docker-compose as default")
		}
		// sure, dockerfile and dockerImage shouldn't be empty here but it's just here for correctness
		if dockerfile != "" || dockerImage != "" || dockerContextDir != "" {
			return nil, fmt.Errorf("Parameters --%s, --%s, %s cannot be used in docker-compose scenario",
				constants.ArgNameDockerfile, constants.ArgNameDockerImage, constants.ArgNameDockerContextDir)
		}
		target = commands.NewDockerComposeBuild(composeFile, composeProjectDir, buildArgs, buildSecretArgs)
	}

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
		k, v, err := parseAssignment(env)
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

func parseAssignment(in string) (string, string, error) {
	values := strings.SplitN(in, "=", 2)
	if len(values) != 2 {
		return "", "", fmt.Errorf("%s cannot be split into 2 tokens with '='", in)
	}
	return values[0], values[1], nil
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
		// when all builds and pushs (if any) succeed
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
