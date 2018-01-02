package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/acr-builder/pkg/constants"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/workflow"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var stockImgDependencies1 = []build.ImageDependencies{
	*testCommon.DependenciesWithDigests(*testCommon.NewImageDependencies("img1", "run1", []string{})),
	*testCommon.DependenciesWithDigests(*testCommon.NewImageDependencies("img1.2", "run1.2", []string{"build1.2"})),
}

var stockImgDependencies2 = []build.ImageDependencies{
	*testCommon.DependenciesWithDigests(*testCommon.NewImageDependencies("img2", "run2", []string{"build2", "build2.1"})),
}

var stockImgDependencies3 = []build.ImageDependencies{
	*testCommon.DependenciesWithDigests(*testCommon.NewImageDependencies("img3", "run3", []string{"build3", "build3.1"})),
}

type dependenciesTestCase struct {
	baseline             []build.ImageDependencies
	new                  []build.ImageDependencies
	err                  error
	expectedDependencies []build.ImageDependencies
	expectedError        string
}

func TestDependenciesTaskEmpty(t *testing.T) {
	// no error, no dependencies returned (should not happen)
	testDependencies(t, dependenciesTestCase{})
}

func TestDependenciesTaskAppend1(t *testing.T) {
	testDependencies(t, dependenciesTestCase{
		baseline:             stockImgDependencies1,
		new:                  stockImgDependencies2,
		expectedDependencies: append(stockImgDependencies1, stockImgDependencies2...),
	})
}

func TestDependenciesTaskAppend2(t *testing.T) {
	testDependencies(t, dependenciesTestCase{
		new:                  stockImgDependencies1,
		expectedDependencies: stockImgDependencies1,
	})
}

func TestDependenciesTaskError(t *testing.T) {
	testDependencies(t, dependenciesTestCase{
		baseline:      stockImgDependencies1,
		new:           stockImgDependencies2,
		err:           fmt.Errorf("boom boom boom"),
		expectedError: "^boom boom boom$",
	})
}

func testDependencies(t *testing.T, tc dependenciesTestCase) {
	runner := new(test.MockRunner)
	defer runner.AssertExpectations(t)
	runner.UseDefaultFileSystem()
	buildTarget := new(test.MockBuildTarget)
	buildTarget.On("ScanForDependencies", runner).Return(tc.new, tc.err).Once()
	outputs := &workflow.OutputContext{
		ImageDependencies: tc.baseline,
	}
	task := dependencyTask(buildTarget)
	err := task(runner, outputs)
	buildTarget.AssertExpectations(t)
	runner.AssertExpectations(t)
	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	} else {
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedDependencies, outputs.ImageDependencies)
	}
}

type buildParameters struct {
	dependencies []build.ImageDependencies
	envVar       []build.EnvVar
	expectedEnv  []string
}

type sourceParameters struct {
	envVar      []build.EnvVar
	builds      []buildParameters
	expectedEnv []string
}

type dockerCredsParameters struct {
	expectedEnv []string
}

type compileTestCase struct {
	buildNumber          string
	registry             string
	userDefined          []build.EnvVar
	creds                []dockerCredsParameters
	sources              []sourceParameters
	push                 bool
	expectedDependencies []build.ImageDependencies
	queryCmdErr          map[string]error
}

func newMultiSourceTestCase(push bool) *compileTestCase {
	gen := testCommon.NewMappedGenerator("value")
	return &compileTestCase{
		buildNumber: "TestCompileHappy-01",
		registry:    "TestCompileHappy.azurecr.io",
		userDefined: []build.EnvVar{
			{Name: "k1", Value: gen.NextWithKey("k1")},
			{Name: "k2", Value: gen.NextWithKey("k2")},
		},
		creds: []dockerCredsParameters{
			{expectedEnv: []string{
				constants.ExportsBuildNumber + "=TestCompileHappy-01",
				constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
				constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
				"k1=" + gen.Lookup("k1"),
				"k2=" + gen.Lookup("k2"),
			}},
		},
		push: push,
		sources: []sourceParameters{
			{
				envVar: []build.EnvVar{
					{Name: "s1.1", Value: gen.NextWithKey("s1.1")},
					{Name: "s1.2", Value: gen.NextWithKey("s1.2")},
				},
				expectedEnv: []string{
					constants.ExportsBuildNumber + "=TestCompileHappy-01",
					constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
					constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
					"k1=" + gen.Lookup("k1"),
					"k2=" + gen.Lookup("k2"),
					"s1.1=" + gen.Lookup("s1.1"),
					"s1.2=" + gen.Lookup("s1.2"),
				},
				builds: []buildParameters{
					{dependencies: stockImgDependencies1,
						envVar: []build.EnvVar{
							{Name: "b1.1.1", Value: gen.NextWithKey("b1.1.1")},
							{Name: "b1.1.2", Value: gen.NextWithKey("b1.1.2")},
						},
						expectedEnv: []string{
							constants.ExportsBuildNumber + "=TestCompileHappy-01",
							constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
							constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
							"k1=" + gen.Lookup("k1"),
							"k2=" + gen.Lookup("k2"),
							"s1.1=" + gen.Lookup("s1.1"),
							"s1.2=" + gen.Lookup("s1.2"),
							"b1.1.1=" + gen.Lookup("b1.1.1"),
							"b1.1.2=" + gen.Lookup("b1.1.2"),
						},
					},
					{dependencies: stockImgDependencies2,
						envVar: []build.EnvVar{
							{Name: "b1.2.1", Value: gen.NextWithKey("b1.2.1")},
							{Name: "b1.2.2", Value: gen.NextWithKey("b1.2.2")},
						},
						expectedEnv: []string{
							constants.ExportsBuildNumber + "=TestCompileHappy-01",
							constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
							constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
							"k1=" + gen.Lookup("k1"),
							"k2=" + gen.Lookup("k2"),
							"s1.1=" + gen.Lookup("s1.1"),
							"s1.2=" + gen.Lookup("s1.2"),
							"b1.2.1=" + gen.Lookup("b1.2.1"),
							"b1.2.2=" + gen.Lookup("b1.2.2"),
						},
					},
				},
			},
			{
				envVar: []build.EnvVar{
					{Name: "s2.1", Value: gen.NextWithKey("s2.1")},
					{Name: "s2.2", Value: gen.NextWithKey("s2.2")},
				},
				expectedEnv: []string{
					constants.ExportsBuildNumber + "=TestCompileHappy-01",
					constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
					constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
					"k1=" + gen.Lookup("k1"),
					"k2=" + gen.Lookup("k2"),
					"s2.1=" + gen.Lookup("s2.1"),
					"s2.2=" + gen.Lookup("s2.2"),
				},
				builds: []buildParameters{
					{dependencies: stockImgDependencies3,
						envVar: []build.EnvVar{
							{Name: "b2.1.1", Value: gen.NextWithKey("b2.1.1")},
							{Name: "b2.1.2", Value: gen.NextWithKey("b2.1.2")},
						},
						expectedEnv: []string{
							constants.ExportsBuildNumber + "=TestCompileHappy-01",
							constants.ExportsDockerRegistry + "=TestCompileHappy.azurecr.io",
							constants.ExportsPushOnSuccess + "=" + strconv.FormatBool(push),
							"k1=" + gen.Lookup("k1"),
							"k2=" + gen.Lookup("k2"),
							"s2.1=" + gen.Lookup("s2.1"),
							"s2.2=" + gen.Lookup("s2.2"),
							"b2.1.1=" + gen.Lookup("b2.1.1"),
							"b2.1.2=" + gen.Lookup("b2.1.2"),
						},
					},
				},
			},
		},
		expectedDependencies: append(append(stockImgDependencies1, stockImgDependencies2...), stockImgDependencies3...),
	}
}

func TestCompileHappy(t *testing.T) {
	testCompile(t, newMultiSourceTestCase(true))
}

func TestCompileNoPush(t *testing.T) {
	testCompile(t, newMultiSourceTestCase(false))
}

func testCompile(t *testing.T, tc *compileTestCase) {
	runner := new(test.MockRunner)
	runner.UseDefaultFileSystem()
	req := &build.Request{
		DockerRegistry: tc.registry,
	}
	for _, cred := range tc.creds {
		credMock := new(test.MockDockerCredential)
		req.DockerCredentials = append(req.DockerCredentials, credMock)
		verifyContext(t, credMock.On("Authenticate", runner), cred.expectedEnv, nil)
		defer credMock.AssertExpectations(t)
	}
	for _, source := range tc.sources {
		builds := []build.Target{}
		for i := range source.builds {
			build := source.builds[i] // do not use foreach loop here because `build` will be used in a closure
			buildMock := new(test.MockBuildTarget)
			verifyContext(t, buildMock.On("Build", runner), build.expectedEnv, nil)
			buildMock.On("Export").Return(build.envVar).Once()
			scanDependencies := buildMock.On("ScanForDependencies", runner, mock.Anything)
			scanDependencies.Run(func(arg mock.Arguments) {
				verifyContextFromParameters(t, build.expectedEnv, arg)
				scanDependencies.ReturnArguments = []interface{}{build.dependencies, nil}
			}).Once()
			if tc.push {
				verifyContext(t, buildMock.On("Push", runner), build.expectedEnv, nil)
			}
			builds = append(builds, buildMock)
			defer buildMock.AssertExpectations(t)
		}
		sourceMock := new(test.MockBuildSource)
		verifyContext(t, sourceMock.On("Obtain", runner), source.expectedEnv, nil)
		verifyContext(t, sourceMock.On("Return", runner), source.expectedEnv, nil)
		sourceMock.On("Export").Return(source.envVar).Once()
		defer sourceMock.AssertExpectations(t)
		target := build.SourceTarget{
			Source: sourceMock,
			Builds: builds,
		}
		req.Targets = append(req.Targets, target)
	}
	runner.PrepareDigestQuery(tc.expectedDependencies, tc.queryCmdErr)
	workflow := compileWorkflow(tc.buildNumber, tc.userDefined, req, tc.push)
	err := workflow.Run(runner)
	assert.Nil(t, err)
	outputs := workflow.GetOutputs()
	testCommon.AssertSameDependencies(t, tc.expectedDependencies, outputs.ImageDependencies)
}

func verifyContextFromParameters(t *testing.T, expected []string, arg mock.Arguments) {
	runner, ok := arg[0].(build.Runner)
	assert.True(t, ok, "Cannot cast input to runner %s", reflect.TypeOf(arg[0]).Name)
	env := runner.GetContext()
	assertSameContext(t, expected, env)
}

func verifyContext(t *testing.T, call *mock.Call, expected []string, rtn error) *mock.Call {
	return call.Run(func(arg mock.Arguments) {
		verifyContextFromParameters(t, expected, arg)
		call.ReturnArguments = []interface{}{rtn}
	}).Once()
}

func assertSameContext(t *testing.T, expected []string, actual *build.BuilderContext) {
	actualEnv := map[string]bool{}
	timeStampFound := false
	for _, entry := range actual.Export() {
		k, v, err := parseAssignment(entry)
		assert.Nil(t, err)
		if k == constants.ExportsBuildTimestamp {
			timeStampFound = true
			buildTime, err := time.Parse(constants.TimestampFormat, v)
			assert.Nil(t, err, "Build time format incorrect")
			assert.WithinDuration(t, time.Now(), buildTime, time.Second*1)
		} else {
			actualEnv[entry] = true
		}
	}
	assert.True(t, timeStampFound)
	expectedButNotFound := []string{}
	for _, entry := range expected {
		if actualEnv[entry] {
			delete(actualEnv, entry)
		} else {
			expectedButNotFound = append(expectedButNotFound, entry)
		}
	}
	assert.Empty(t, expectedButNotFound, "Expected entries not found %v", expectedButNotFound)
	assert.Empty(t, actualEnv, "Actual entries not expected %v", reflect.ValueOf(actualEnv).MapKeys())
}

type parseUserDefinedTestCase struct {
	input         []string
	expected      []build.EnvVar
	expectedError string
}

func TestParseUserDefinedEmpty(t *testing.T) {
	testParseUserDefined(t, parseUserDefinedTestCase{})
}

func TestParseUserDefinedSuccess(t *testing.T) {
	testParseUserDefined(t, parseUserDefinedTestCase{
		input: []string{
			"hello=world",
			"foo=bar",
		},
		expected: []build.EnvVar{
			{Name: "hello", Value: "world"},
			{Name: "foo", Value: "bar"},
		},
	})
}

func TestParseUserDefinedFail(t *testing.T) {
	testParseUserDefined(t, parseUserDefinedTestCase{
		input: []string{
			"hello=world",
			"0hello=0world",
		},
		expectedError: "^Invalid environmental variable name: 0hello$",
	})
}

func TestParseUserDefinedFail2(t *testing.T) {
	testParseUserDefined(t, parseUserDefinedTestCase{
		input: []string{
			"hello world",
		},
		expectedError: "^Error parsing build environment \"hello world cannot be split into 2 tokens with '='\"$",
	})
}

func testParseUserDefined(t *testing.T, tc parseUserDefinedTestCase) {
	env, err := parseUserDefined(tc.input)
	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	} else {
		assert.Nil(t, err)
		testCommon.AssertSameEnv(t, tc.expected, env)
	}
}

type createBuildRequestTestCase struct {
	composeFile       string
	composeProjectDir string
	dockerfile        string
	dockerImage       string
	dockerContextDir  string
	dockerUser        string
	dockerPW          string
	dockerRegistry    string
	workingDir        string
	gitURL            string
	gitBranch         string
	gitHeadRev        string
	gitPATokenUser    string
	gitPAToken        string
	gitXToken         string
	webArchive        string
	buildArgs         []string
	push              bool
	files             test.FileSystemExpectations
	expected          build.Request
	expectedError     string
}

func TestCreateBuildRequestNoParams(t *testing.T) {
	localSource := commands.NewLocalSource("")
	testCreateBuildRequest(t, createBuildRequestTestCase{
		expected: build.Request{
			DockerCredentials: []build.DockerCredential{},
			Targets: []build.SourceTarget{
				{
					Source: localSource,
					Builds: []build.Target{commands.NewDockerComposeBuild("", "", nil)},
				},
			},
		},
	})
}

func TestCreateBuildRequestWithGitPATokenDockerfile(t *testing.T) {
	gitUser := "some.git.user"
	gitPassword := "some.git.pw"
	gitAddress := "some.git-repository"
	branch := "some.branch"
	headRev := "some.rev"
	targetDir := "some.dir"
	dockerfile := "some.dockerfile"
	contextDir := "some.contextDir"
	buildArgs := []string{"k1=v1", "k2=v2"}
	registry := "some.registry"
	imageName := "some.image"
	dockerBuildTarget := commands.NewDockerBuild(dockerfile, contextDir,
		buildArgs, registry+"/", imageName)
	gitCred, err := commands.NewGitPersonalAccessToken(gitUser, gitPassword)
	assert.Nil(t, err)
	gitSource := commands.NewGitSource(gitAddress, branch, headRev, targetDir, gitCred)
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitPATokenUser:   gitUser,
		gitPAToken:       gitPassword,
		gitURL:           gitAddress,
		gitBranch:        branch,
		gitHeadRev:       headRev,
		workingDir:       targetDir,
		dockerfile:       dockerfile,
		dockerContextDir: contextDir,
		buildArgs:        buildArgs,
		dockerRegistry:   registry,
		dockerImage:      imageName,
		expected: build.Request{
			DockerRegistry:    registry + "/",
			DockerCredentials: []build.DockerCredential{},
			Targets: []build.SourceTarget{
				{
					Source: gitSource,
					Builds: []build.Target{dockerBuildTarget},
				},
			},
		},
	})
}

func TestCreateBuildRequestWithGitXTokenDockerCompose(t *testing.T) {
	dockerRegistry := "some.dockerRegistry"
	dockerUser := "some.dockerUser"
	dockerPW := "some.********"
	gitXToken := "some.x.token"
	gitAddress := "some.git-repository"
	branch := "some.branch"
	headRev := "some.rev"
	targetDir := "some.dir"
	composeFile := "some.composefile"
	composeProjectDir := "some.projectdir"
	buildArgs := []string{"k3=v3", "k4=v4"}
	cred, err := commands.NewDockerUsernamePassword(dockerRegistry, dockerUser, dockerPW)
	assert.Nil(t, err)
	gitSource := commands.NewGitSource(gitAddress, branch, headRev, targetDir, commands.NewGitXToken(gitXToken))
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerRegistry:    dockerRegistry,
		dockerUser:        dockerUser,
		dockerPW:          dockerPW,
		gitXToken:         gitXToken,
		gitURL:            gitAddress,
		gitBranch:         branch,
		gitHeadRev:        headRev,
		workingDir:        targetDir,
		composeFile:       composeFile,
		composeProjectDir: composeProjectDir,
		buildArgs:         buildArgs,
		expected: build.Request{
			DockerRegistry:    dockerRegistry + "/",
			DockerCredentials: []build.DockerCredential{cred},
			Targets: []build.SourceTarget{
				{
					Source: gitSource,
					Builds: []build.Target{commands.NewDockerComposeBuild(composeFile, composeProjectDir, buildArgs)},
				},
			},
		},
	})
}

func TestCreateBuildRequestNoDockerPassword(t *testing.T) {
	dockerRegistry := "some.dockerRegistry"
	dockerUser := "some.dockerUser"
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerRegistry: dockerRegistry,
		dockerUser:     dockerUser,
		expectedError: fmt.Sprintf("^Please provide both --%s and --%s or neither$",
			constants.ArgNameDockerUser, constants.ArgNameDockerPW),
	})
}

func TestCreateBuildRequestNoGitPassword(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitURL:         "some.git.url",
		gitPATokenUser: "some.git.user",
		expectedError: fmt.Sprintf("^Please provide both --%s and --%s or neither$",
			constants.ArgNameGitPATokenUser, constants.ArgNameGitPAToken),
	})
}

func TestCreateBuildRequestNoGitURL(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitPATokenUser: "some.git.user",
		expectedError: fmt.Sprintf("^Optional parameter %s is given for %s but none of the required parameters: \\[%s\\] were given$",
			constants.ArgNameGitPATokenUser, constants.SourceNameGit, constants.ArgNameGitURL),
	})
}

func TestCreateBuildRequestDockerImageDefinedConflictingParameters(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerImage:       "someImage",
		composeProjectDir: "some.compose.dir",
		expectedError:     fmt.Sprintf("^Parameter --%s cannot be used for dockerfile build scenario$", constants.ArgNameDockerComposeProjectDir),
	})
}

func TestCreateBuildRequestDockerComposeConflictingParameters(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		composeFile:      "docker-compose.yml",
		dockerContextDir: "some.dir",
		expectedError: fmt.Sprintf("Parameters --%s, --%s, %s cannot be used in docker-compose scenario",
			constants.ArgNameDockerfile, constants.ArgNameDockerImage, constants.ArgNameDockerContextDir),
	})
}

func TestCreateBuildRequestDockerBuildCreationError(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerfile: "some.dockerfile",
		push:       true,
		expectedError: fmt.Sprintf("^Docker registry is needed for push, use --%s or environment variable %s to provide its value$",
			constants.ArgNameDockerRegistry, constants.ExportsDockerRegistry),
	})
}

func testCreateBuildRequest(t *testing.T, tc createBuildRequestTestCase) {
	runner := test.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test.MockFileSystem)
	fs.PrepareFileSystem(tc.files)
	defer fs.AssertExpectations(t)
	builder := NewBuilder(runner)
	req, err := builder.createBuildRequest(tc.composeFile, tc.composeProjectDir,
		tc.dockerfile, tc.dockerImage, tc.dockerContextDir,
		tc.dockerUser, tc.dockerPW, tc.dockerRegistry, tc.workingDir,
		tc.gitURL, tc.gitBranch, tc.gitHeadRev,
		tc.gitPATokenUser, tc.gitPAToken, tc.gitXToken,
		tc.webArchive, tc.buildArgs, tc.push)

	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	} else {
		assert.Nil(t, err)
		assert.Equal(t, tc.expected, *req)
	}
}

type runTestCase struct {
	buildNumber          string
	composeFile          string
	composeProjectDir    string
	dockerfile           string
	dockerImage          string
	dockerContextDir     string
	dockerUser           string
	dockerPW             string
	dockerRegistry       string
	workingDir           string
	gitURL               string
	gitBranch            string
	gitHeadRev           string
	gitPATokenUser       string
	gitPAToken           string
	gitXToken            string
	webArchive           string
	buildEnvs            []string
	buildArgs            []string
	push                 bool
	expectedCommands     []test.CommandsExpectation
	expectedDependencies []build.ImageDependencies
	queryCmdErr          map[string]error
	expectedErr          string
}

func TestRunSimpleHappy(t *testing.T) {
	testRun(t, runTestCase{
		buildNumber:    "buildNum-0",
		dockerRegistry: testCommon.TestsDockerRegistryName,
		workingDir:     filepath.Join("..", "..", "tests", "resources", "docker-compose"),
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"build"},
			},
		},
		expectedDependencies: []build.ImageDependencies{
			*testCommon.DependenciesWithDigests(testCommon.MultistageExampleDependencies),
			*testCommon.DependenciesWithDigests(testCommon.HelloNodeExampleDependencies),
		},
	})
}

func TestRunNoRegistryGiven(t *testing.T) {
	os.Clearenv()
	testRun(t, runTestCase{
		buildNumber: "buildNum-0",
		workingDir:  filepath.Join("..", "..", "tests", "resources", "docker-compose"),
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"build"},
			},
		},
		expectedDependencies: []build.ImageDependencies{
			*testCommon.DependenciesWithDigests(testCommon.MultistageExampleDependenciesOn("")),
			*testCommon.DependenciesWithDigests(testCommon.HelloNodeExampleDependenciesOn("")),
		},
	})
}

func TestRunNoRegistryGivenPush(t *testing.T) {
	os.Clearenv()
	testRun(t, runTestCase{
		push:        true,
		buildNumber: "buildNum-0",
		workingDir:  filepath.Join("..", "..", "tests", "resources", "docker-compose"),
		expectedErr: fmt.Sprintf("^Docker registry is needed for push, use --%s or environment variable %s to provide its value$",
			constants.ArgNameDockerRegistry, constants.ExportsDockerRegistry),
	})
}

func TestRunParseEnvFailed(t *testing.T) {
	testRun(t, runTestCase{
		dockerRegistry: testCommon.DotnetExampleTargetRegistryName,
		buildEnvs:      []string{"*invalid=value"},
		expectedErr:    "^Invalid environmental variable name: \\*invalid$",
	})
}

func TestCreateBuildRequestFailed(t *testing.T) {
	testRun(t, runTestCase{
		dockerRegistry: testCommon.DotnetExampleTargetRegistryName,
		dockerUser:     "someUser",
		expectedErr: fmt.Sprintf("^Please provide both --%s and --%s or neither$",
			constants.ArgNameDockerUser, constants.ArgNameDockerPW),
	})
}

func testRun(t *testing.T, tc runTestCase) {
	runner := test.NewMockRunner()
	defer runner.AssertExpectations(t)
	runner.UseDefaultFileSystem()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	runner.PrepareDigestQuery(tc.expectedDependencies, tc.queryCmdErr)
	builder := NewBuilder(runner)
	startTime := time.Now()
	dependencies, duration, err := builder.Run(tc.buildNumber, tc.composeFile, tc.composeProjectDir,
		tc.dockerfile, tc.dockerImage, tc.dockerContextDir,
		tc.dockerUser, tc.dockerPW, tc.dockerRegistry,
		tc.workingDir, tc.gitURL, tc.gitBranch, tc.gitHeadRev,
		tc.gitPATokenUser, tc.gitPAToken, tc.gitXToken,
		tc.webArchive, tc.buildEnvs, tc.buildArgs, tc.push)
	actualDuration := time.Since(startTime)
	assert.True(t, actualDuration >= duration)
	assert.True(t, duration+time.Millisecond >= actualDuration)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
		testCommon.AssertSameDependencies(t, tc.expectedDependencies, dependencies)
	}
}
