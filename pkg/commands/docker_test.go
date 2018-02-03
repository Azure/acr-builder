package commands

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewDockerUsernamePasswordFailed(t *testing.T) {
	_, err := NewDockerUsernamePassword("registry", "", "password")
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Errorf("Please provide both --%s and --%s or neither", constants.ArgNameDockerUser, constants.ArgNameDockerPW), err)
}

func TestDockerUsernamePasswordAuthenticate(t *testing.T) {
	m, err := NewDockerUsernamePassword("registry", "username", "password")
	assert.Nil(t, err)
	runner := test.NewMockRunner()
	parameters := []string{"login", "-u", "username", "-p", "password", "registry"}
	call := runner.On("ExecuteCmdWithObfuscation",
		mock.Anything,
		"docker",
		parameters).Times(1)
	call.Run(func(args mock.Arguments) {
		assert.NotNil(t, args[0])
		obfFunc, ok := args[0].(func([]string))
		assert.True(t, ok)
		obfFunc(parameters)
		assert.Equal(t, []string{"login", "-u", "username", "-p", constants.ObfuscationString, "registry"}, parameters)
		// this part is just to prove that things do not blow up on invalid data
		invalidData := []string{"-p"}
		obfFunc(invalidData)
		assert.Equal(t, []string{"-p"}, invalidData)
		call.ReturnArguments = []interface{}{fmt.Errorf("some error")}
	})
	err = m.Authenticate(runner)
	assert.NotNil(t, err)
	assert.Equal(t, "some error", err.Error())
}

type dockerTestCase struct {
	dockerfile           string
	contextDir           string
	buildArgs            []string
	registry             string
	imageName            string
	expectedCommands     []test.CommandsExpectation
	expectedExecutionErr string
}

func testDockerBuild(t *testing.T, tc dockerTestCase) {
	runner := new(test.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	target := NewDockerBuild(tc.dockerfile, tc.contextDir, tc.buildArgs, tc.registry, tc.imageName)
	err := target.Build(runner)
	if tc.expectedExecutionErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedExecutionErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerBuildMinimalParametersFailedEventually(t *testing.T) {
	// minimal (zero) parameters
	// dockerfile is not there, dependencies resolution will fail but build will go on
	// let's return a dependency error for fun
	testDockerBuild(t, dockerTestCase{
		expectedCommands: []test.CommandsExpectation{
			{
				Command:  "docker",
				Args:     []string{"build", "."},
				ErrorMsg: "docker build error",
			},
		},
		expectedExecutionErr: "^docker build error$",
	})
}

func TestDockerBuildHappy(t *testing.T) {
	// minimal (zero) parameters
	// dockerfile is not there, dependencies resolution will fail but build will go on
	// let's return a dependency error for fun
	testDockerBuild(t, dockerTestCase{
		dockerfile: filepath.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile"),
		contextDir: "contextDir",
		buildArgs:  []string{"k1=v1", "k2=v2"},
		registry:   testCommon.DotnetExampleTargetRegistryName + "/",
		imageName:  testCommon.DotnetExampleTargetImageName,
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "docker",
				Args: []string{"build", "-f", filepath.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile"),
					"-t", testCommon.DotnetExampleFullImageName, "--build-arg", "k1=v1", "--build-arg", "k2=v2", "contextDir"},
			},
		},
	})
}

func TestExport(t *testing.T) {
	task := NewDockerBuild("myDockerfile", "myContextDir", []string{}, "myRegistry/", "myImage")
	exports := task.Export()
	testCommon.AssertSameEnv(t, []build.EnvVar{
		{Name: constants.ExportsDockerfilePath, Value: "myDockerfile"},
		{Name: constants.ExportsDockerBuildContext, Value: "myContextDir"},
		{Name: constants.ExportsDockerPushImage, Value: "myRegistry/myImage"},
	}, exports)
}

func testDockerPush(t *testing.T, tc dockerTestCase) {
	runner := new(test.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	target := NewDockerBuild(tc.dockerfile, tc.contextDir, tc.buildArgs, tc.registry, tc.imageName)
	err := target.Push(runner)
	if tc.expectedExecutionErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedExecutionErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerPushNoTarget(t *testing.T) {
	testDockerPush(t, dockerTestCase{
		expectedExecutionErr: "^No push target is defined$",
	})
}

func TestDockerPushHappy(t *testing.T) {
	testDockerPush(t, dockerTestCase{
		registry:  "myRegistry/",
		imageName: "myImage",
		expectedCommands: []test.CommandsExpectation{{
			Command: "docker",
			Args:    []string{"push", "myRegistry/myImage"},
		}},
	})
}

type dockerDependenciesTestCase struct {
	path        string
	expectedErr string
}

func TestDockerScanDependenciesHappy(t *testing.T) {
	testDockerScanDependencies(t, dockerDependenciesTestCase{
		path: filepath.Join("$project_root", "Dockerfile"),
	})
}

func TestDockerScanDependenciesError(t *testing.T) {
	testDockerScanDependencies(t, dockerDependenciesTestCase{
		expectedErr: "^Error opening dockerfile Dockerfile",
	})
}

func testDockerScanDependencies(t *testing.T, tc dockerDependenciesTestCase) {
	runner := new(test.MockRunner)
	defer runner.AssertExpectations(t)
	runner.SetContext(build.NewContext([]build.EnvVar{
		{Name: "project_root", Value: filepath.Join("..", "..", "tests", "resources", "docker-dotnet")},
	}, []build.EnvVar{}))
	target := NewDockerBuild(tc.path, "", []string{}, testCommon.DotnetExampleTargetRegistryName+"/", testCommon.DotnetExampleTargetImageName)
	dependencies, err := target.ScanForDependencies(runner)
	if tc.expectedErr == "" {
		assert.Nil(t, err)
		testCommon.AssertSameDependencies(t, []build.ImageDependencies{testCommon.DotnetExampleDependencies}, dependencies)
	} else {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	}
}

type repoDigestTestcase struct {
	jsonContent    string
	reference      *build.ImageReference
	expectedDigest string
}

func TestDockerGetRepoDigestSucceed(t *testing.T) {
	testDockerGetRepoDigest(t, repoDigestTestcase{
		jsonContent: "[\"registry2/repo2@sha256:testsha2\", \"registry1/repo1@sha256:testsha1\"]",
		reference: &build.ImageReference{
			Registry:   "registry1",
			Repository: "repo1",
		},
		expectedDigest: "sha256:testsha1",
	})

	testDockerGetRepoDigest(t, repoDigestTestcase{
		jsonContent: "[\"repo3@sha256:testsha3\"]",
		reference: &build.ImageReference{
			Registry:   "",
			Repository: "repo3",
		},
		expectedDigest: "sha256:testsha3",
	})
}

func TestDockerGetRepoDigestFailed(t *testing.T) {
	testDockerGetRepoDigest(t, repoDigestTestcase{
		jsonContent: "invalidjson",
		reference: &build.ImageReference{
			Registry:   "registry1",
			Repository: "repo1",
		},
		expectedDigest: "",
	})

	testDockerGetRepoDigest(t, repoDigestTestcase{
		jsonContent: "[\"registry2/repo2@sha256:testsha2\", \"registry1/repo1@sha256:testsha1\"]",
		reference: &build.ImageReference{
			Registry:   "registry3",
			Repository: "repo3",
		},
		expectedDigest: "",
	})
}

func testDockerGetRepoDigest(t *testing.T, tc repoDigestTestcase) {
	actualDigest := getRepoDigest(tc.jsonContent, tc.reference)
	assert.Equal(t, tc.expectedDigest, actualDigest)
}
