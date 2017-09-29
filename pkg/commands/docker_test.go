package commands

import (
	"fmt"
	"path"
	"regexp"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	test_domain "github.com/Azure/acr-builder/tests/mocks/pkg/domain"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
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
	runner := test_domain.NewMockRunner()
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
	push                 bool
	registry             string
	imageName            string
	expectedCommands     []test_domain.CommandsExpectation
	expectedCreationErr  string
	expectedExecutionErr string
}

func testDockerBuild(t *testing.T, tc dockerTestCase) {
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	target, err := NewDockerBuild(tc.dockerfile, tc.contextDir, tc.buildArgs, tc.push, tc.registry, tc.imageName)
	if tc.expectedCreationErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedCreationErr), err.Error())
		return
	}
	err = target.Build(runner)
	if tc.expectedExecutionErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedExecutionErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerBuildTaskExecuteNoImage(t *testing.T) {
	testDockerBuild(t, dockerTestCase{
		push:                true,
		expectedCreationErr: fmt.Sprintf("^When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage),
	})
}

func TestDockerBuildMinimalParametersFailedEventually(t *testing.T) {
	// minimal (zero) parameters
	// dockerfile is not there, dependencies resolution will fail but build will go on
	// let's return a dependency error for fun
	testDockerBuild(t, dockerTestCase{
		expectedCommands: []test_domain.CommandsExpectation{
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
		dockerfile: path.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile"),
		contextDir: "contextDir",
		buildArgs:  []string{"k1=v1", "k2=v2"},
		registry:   testutils.DotnetExampleTargetRegistryName,
		imageName:  testutils.DotnetExampleTargetImageName,
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker",
				Args:    []string{"build", "-f", "../../tests/resources/docker-dotnet/Dockerfile", "-t", testutils.DotnetExampleDependencies.Image, "--build-arg", "k1=v1", "--build-arg", "k2=v2", "contextDir"},
			},
		},
	})
}

func TestExport(t *testing.T) {
	task, err := NewDockerBuild("myDockerfile", "myContextDir", []string{}, false, "myRegistry", "myImage")
	assert.Nil(t, err)
	exports := task.Export()
	assert.Equal(t, 3, len(exports))
	for _, entry := range exports {
		switch entry.Name {
		case constants.DockerfilePathVar:
			assert.Equal(t, "myDockerfile", entry.Value)
		case constants.DockerBuildContextVar:
			assert.Equal(t, "myContextDir", entry.Value)
		case constants.DockerPushImageVar:
			assert.Equal(t, "myRegistry/myImage", entry.Value)
		default:
			assert.Fail(t, "unexpected entry: %s: %s", entry.Name, entry.Value)
		}
	}
}

func testDockerPush(t *testing.T, tc dockerTestCase) {
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	target, err := NewDockerBuild(tc.dockerfile, tc.contextDir, tc.buildArgs, tc.push, tc.registry, tc.imageName)
	err = target.Push(runner)
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
		registry:  "myRegistry",
		imageName: "myImage",
		expectedCommands: []test_domain.CommandsExpectation{{
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
		path: path.Join("$project_root", "Dockerfile"),
	})
}

func TestDockerScanDependenciesError(t *testing.T) {
	testDockerScanDependencies(t, dockerDependenciesTestCase{
		expectedErr: "^Error opening dockerfile Dockerfile",
	})
}

func testDockerScanDependencies(t *testing.T, tc dockerDependenciesTestCase) {
	runner := new(test_domain.MockRunner)
	defer runner.AssertExpectations(t)
	runner.SetContext(domain.NewContext([]domain.EnvVar{
		{Name: "project_root", Value: path.Join("..", "..", "tests", "resources", "docker-dotnet")},
	}, []domain.EnvVar{}))
	target, err := NewDockerBuild(tc.path, "", []string{}, false,
		testutils.DotnetExampleTargetRegistryName, testutils.DotnetExampleTargetImageName)
	assert.Nil(t, err)
	dependencies, err := target.ScanForDependencies(runner)
	if tc.expectedErr == "" {
		assert.Nil(t, err)
		testutils.AssertSameDependencies(t, []domain.ImageDependencies{testutils.DotnetExampleDependencies}, dependencies)
	} else {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	}
}
