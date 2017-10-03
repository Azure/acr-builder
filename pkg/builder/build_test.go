package builder

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"

	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/pkg/workflow"
	test_domain "github.com/Azure/acr-builder/tests/mocks/pkg/domain"
	"github.com/Azure/acr-builder/tests/testCommon"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var stockImgDependencies1 = []domain.ImageDependencies{
	{
		Image:             "img1",
		RuntimeDependency: "run1",
		BuildDependencies: []string{},
	},
	{
		Image:             "img1.2",
		RuntimeDependency: "run1.2",
		BuildDependencies: []string{"build1.2"},
	},
}

var stockImgDependencies2 = []domain.ImageDependencies{
	{
		Image:             "img2",
		RuntimeDependency: "run2",
		BuildDependencies: []string{"build2", "build2.1"},
	},
}

var stockImgDependencies3 = []domain.ImageDependencies{
	{
		Image:             "img3",
		RuntimeDependency: "run3",
		BuildDependencies: []string{"build3", "build3.1"},
	},
}

type dependenciesTestCase struct {
	baseline             []domain.ImageDependencies
	new                  []domain.ImageDependencies
	err                  error
	expectedDependencies []domain.ImageDependencies
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
		baseline:             stockImgDependencies1,
		new:                  stockImgDependencies2,
		err:                  fmt.Errorf("boom boom boom"),
		expectedDependencies: stockImgDependencies1,
	})
}

func testDependencies(t *testing.T, tc dependenciesTestCase) {
	runner := new(test_domain.MockRunner)
	runner.UseDefaultFileSystem()
	buildTarget := new(test_domain.MockBuildTarget)
	buildTarget.On("ScanForDependencies", runner).Return(tc.new, tc.err).Once()
	outputs := &workflow.OutputContext{
		ImageDependencies: tc.baseline,
	}
	task := dependencyTask(buildTarget)
	err := task(runner, outputs)
	buildTarget.AssertExpectations(t)
	runner.AssertExpectations(t)
	assert.Nil(t, err)
	assert.Equal(t, tc.expectedDependencies, outputs.ImageDependencies)
}

type buildParameters struct {
	dependencies []domain.ImageDependencies
	envVar       []domain.EnvVar
	expectedEnv  []string
}

type sourceParameters struct {
	envVar      []domain.EnvVar
	builds      []buildParameters
	expectedEnv []string
}

type dockerCredsParameters struct {
	expectedEnv []string
}

type compileTestCase struct {
	buildNumber          string
	registry             string
	userDefined          []domain.EnvVar
	creds                []dockerCredsParameters
	sources              []sourceParameters
	push                 bool
	expectedDependencies []domain.ImageDependencies
}

func newMultiSourceTestCase(push bool) *compileTestCase {
	gen := testCommon.NewMappedGenerator("value")
	return &compileTestCase{
		buildNumber: "TestCompileHappy-01",
		registry:    "TestCompileHappy.azurecr.io",
		userDefined: []domain.EnvVar{
			{Name: "k1", Value: gen.NextWithKey("k1")},
			{Name: "k2", Value: gen.NextWithKey("k2")},
		},
		creds: []dockerCredsParameters{
			{expectedEnv: []string{
				"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
				"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
				"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
				"k1=" + gen.Lookup("k1"),
				"k2=" + gen.Lookup("k2"),
			}},
		},
		push: push,
		sources: []sourceParameters{
			{
				envVar: []domain.EnvVar{
					{Name: "s1.1", Value: gen.NextWithKey("s1.1")},
					{Name: "s1.2", Value: gen.NextWithKey("s1.2")},
				},
				expectedEnv: []string{
					"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
					"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
					"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
					"k1=" + gen.Lookup("k1"),
					"k2=" + gen.Lookup("k2"),
					"s1.1=" + gen.Lookup("s1.1"),
					"s1.2=" + gen.Lookup("s1.2"),
				},
				builds: []buildParameters{
					{dependencies: stockImgDependencies1,
						envVar: []domain.EnvVar{
							{Name: "b1.1.1", Value: gen.NextWithKey("b1.1.1")},
							{Name: "b1.1.2", Value: gen.NextWithKey("b1.1.2")},
						},
						expectedEnv: []string{
							"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
							"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
							"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
							"k1=" + gen.Lookup("k1"),
							"k2=" + gen.Lookup("k2"),
							"s1.1=" + gen.Lookup("s1.1"),
							"s1.2=" + gen.Lookup("s1.2"),
							"b1.1.1=" + gen.Lookup("b1.1.1"),
							"b1.1.2=" + gen.Lookup("b1.1.2"),
						},
					},
					{dependencies: stockImgDependencies2,
						envVar: []domain.EnvVar{
							{Name: "b1.2.1", Value: gen.NextWithKey("b1.2.1")},
							{Name: "b1.2.2", Value: gen.NextWithKey("b1.2.2")},
						},
						expectedEnv: []string{
							"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
							"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
							"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
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
				envVar: []domain.EnvVar{
					{Name: "s2.1", Value: gen.NextWithKey("s2.1")},
					{Name: "s2.2", Value: gen.NextWithKey("s2.2")},
				},
				expectedEnv: []string{
					"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
					"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
					"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
					"k1=" + gen.Lookup("k1"),
					"k2=" + gen.Lookup("k2"),
					"s2.1=" + gen.Lookup("s2.1"),
					"s2.2=" + gen.Lookup("s2.2"),
				},
				builds: []buildParameters{
					{dependencies: stockImgDependencies3,
						envVar: []domain.EnvVar{
							{Name: "b2.1.1", Value: gen.NextWithKey("b2.1.1")},
							{Name: "b2.1.2", Value: gen.NextWithKey("b2.1.2")},
						},
						expectedEnv: []string{
							"ACR_BUILD_BUILD_NUMBER=TestCompileHappy-01",
							"ACR_BUILD_DOCKER_REGISTRY=TestCompileHappy.azurecr.io",
							"ACR_BUILD_PUSH_ON_SUCCESS=" + strconv.FormatBool(push),
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
	runner := new(test_domain.MockRunner)
	runner.UseDefaultFileSystem()
	req := &domain.BuildRequest{}
	for i := range tc.creds {
		cred := tc.creds[i]
		credMock := new(test_domain.MockDockerCredential)
		req.DockerCredentials = append(req.DockerCredentials, credMock)
		verifyContext(t, credMock.On("Authenticate", runner), cred.expectedEnv, nil)
		defer credMock.AssertExpectations(t)
	}
	for i := range tc.sources {
		source := tc.sources[i]
		builds := []domain.BuildTarget{}
		for j := range source.builds {
			build := source.builds[j]
			buildMock := new(test_domain.MockBuildTarget)
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
		sourceMock := new(test_domain.MockBuildSource)
		verifyContext(t, sourceMock.On("Obtain", runner), source.expectedEnv, nil)
		verifyContext(t, sourceMock.On("Return", runner), source.expectedEnv, nil)
		sourceMock.On("Export").Return(source.envVar).Once()
		defer sourceMock.AssertExpectations(t)
		target := domain.SourceTarget{
			Source: sourceMock,
			Builds: builds,
		}
		req.Targets = append(req.Targets, target)
	}
	workflow := compile(tc.buildNumber, tc.registry, tc.userDefined, req, tc.push)
	err := workflow.Run(runner)
	assert.Nil(t, err)
	outputs := workflow.GetOutputs()
	testutils.AssertSameDependencies(t, tc.expectedDependencies, outputs.ImageDependencies)
}

func verifyContextFromParameters(t *testing.T, expected []string, arg mock.Arguments) {
	runner, ok := arg[0].(domain.Runner)
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

func assertSameContext(t *testing.T, expected []string, actual *domain.BuilderContext) {
	actualEnv := map[string]bool{}
	for _, entry := range actual.Export() {
		actualEnv[entry] = true
	}
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
	expected      []domain.EnvVar
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
		expected: []domain.EnvVar{
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
		testutils.AssertSameEnv(t, tc.expected, env)
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
	gitURL            string
	gitCloneDir       string
	gitBranch         string
	gitHeadRev        string
	gitPATokenUser    string
	gitPAToken        string
	gitXToken         string
	localSource       string
	buildArgs         []string
	push              bool
	files             test_domain.FileSystemExpectations
	expected          domain.BuildRequest
	expectedError     string
}

func TestCreateBuildRequestNoParams(t *testing.T) {
	localSource := commands.NewLocalSource("")
	testCreateBuildRequest(t, createBuildRequestTestCase{
		expected: domain.BuildRequest{
			DockerCredentials: []domain.DockerCredential{},
			Targets: []domain.SourceTarget{
				{
					Source: localSource,
					Builds: []domain.BuildTarget{commands.NewDockerComposeBuild("", "", nil)},
				},
			},
		},
		files: make(test_domain.FileSystemExpectations, 0).AssertFileExists(commands.DockerComposeSupportedFilenames[0], true, nil),
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
	dockerBuildTarget, err := commands.NewDockerBuild(dockerfile, contextDir,
		buildArgs, true, registry, imageName)
	assert.Nil(t, err)
	gitCred, err := commands.NewGitPersonalAccessToken(gitUser, gitPassword)
	assert.Nil(t, err)
	gitSource, err := commands.NewGitSource(gitAddress, branch, headRev, targetDir, gitCred)
	assert.Nil(t, err)
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitPATokenUser:   gitUser,
		gitPAToken:       gitPassword,
		gitURL:           gitAddress,
		gitBranch:        branch,
		gitHeadRev:       headRev,
		gitCloneDir:      targetDir,
		dockerfile:       dockerfile,
		dockerContextDir: contextDir,
		buildArgs:        buildArgs,
		dockerRegistry:   registry,
		dockerImage:      imageName,
		expected: domain.BuildRequest{
			DockerCredentials: []domain.DockerCredential{},
			Targets: []domain.SourceTarget{
				{
					Source: gitSource,
					Builds: []domain.BuildTarget{dockerBuildTarget},
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
	gitSource, err := commands.NewGitSource(gitAddress, branch, headRev, targetDir, commands.NewGitXToken(gitXToken))
	assert.Nil(t, err)
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerRegistry:    dockerRegistry,
		dockerUser:        dockerUser,
		dockerPW:          dockerPW,
		gitXToken:         gitXToken,
		gitURL:            gitAddress,
		gitBranch:         branch,
		gitHeadRev:        headRev,
		gitCloneDir:       targetDir,
		composeFile:       composeFile,
		composeProjectDir: composeProjectDir,
		buildArgs:         buildArgs,
		expected: domain.BuildRequest{
			DockerCredentials: []domain.DockerCredential{cred},
			Targets: []domain.SourceTarget{
				{
					Source: gitSource,
					Builds: []domain.BuildTarget{commands.NewDockerComposeBuild(composeFile, composeProjectDir, buildArgs)},
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

func TestCreateBuildRequestNoGitBranchOrRev(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitURL: "some.git.url",
		expectedError: fmt.Sprintf("^Please provide a --%s or --%s parameter when using git source$",
			constants.ArgNameGitBranch, constants.ArgNameGitHeadRev),
	})
}

func TestCreateBuildRequestNoGitURL(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		gitPATokenUser: "some.git.user",
		expectedError:  fmt.Sprintf("^Git credentials are given but --%s was not$", constants.ArgNameGitURL),
	})
}

func TestCreateBuildRequestErrorFindingDockerComposeFile(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		files: make(test_domain.FileSystemExpectations, 0).
			AssertFileExists(commands.DockerComposeSupportedFilenames[0], false, fmt.Errorf("boom")),
		expectedError: "^Unexpected error while checking for default docker compose file: boom$",
	})
}

func TestCreateBuildRequestDockfileConflictingParameters(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		composeProjectDir: "some.compose.dir",
		files: make(test_domain.FileSystemExpectations, 0).
			AssertFileExists(commands.DockerComposeSupportedFilenames[0], false, nil).
			AssertFileExists(commands.DockerComposeSupportedFilenames[1], false, nil),
		expectedError: fmt.Sprintf("^Parameter --%s cannot be used for dockerfile build scenario$", constants.ArgNameDockerComposeProjectDir),
	})
}

func TestCreateBuildRequestDockerBuildCreationError(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		dockerfile:    "some.dockerfile",
		push:          true,
		expectedError: fmt.Sprintf("^When building with dockerfile, docker image name --%s is required for pushing$", constants.ArgNameDockerImage),
	})
}

func TestCreateBuildRequestComposeFileConflictingParameters(t *testing.T) {
	testCreateBuildRequest(t, createBuildRequestTestCase{
		composeFile: "some.compose.file",
		dockerImage: "some.docker.image",
		expectedError: fmt.Sprintf("^Parameters --%s, --%s, %s cannot be used in docker-compose scenario$",
			constants.ArgNameDockerfile, constants.ArgNameDockerImage, constants.ArgNameDockerContextDir),
	})
}

func testCreateBuildRequest(t *testing.T, tc createBuildRequestTestCase) {
	runner := test_domain.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test_domain.MockFileSystem)
	fs.PrepareFileSystem(tc.files)
	defer fs.AssertExpectations(t)
	req, err := createBuildRequest(runner, tc.composeFile, tc.composeProjectDir,
		tc.dockerfile, tc.dockerImage, tc.dockerContextDir,
		tc.dockerUser, tc.dockerPW, tc.dockerRegistry,
		tc.gitURL, tc.gitCloneDir, tc.gitBranch, tc.gitHeadRev,
		tc.gitPATokenUser, tc.gitPAToken, tc.gitXToken,
		tc.localSource, tc.buildArgs, tc.push)

	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	} else {
		assert.Nil(t, err)
		assert.Equal(t, tc.expected, *req)
	}
}
