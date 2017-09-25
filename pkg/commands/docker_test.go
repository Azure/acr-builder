package commands

import (
	"fmt"
	"path"
	"regexp"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	test_domain "github.com/Azure/acr-builder/tests/utils/domain"
	testutils "github.com/Azure/acr-builder/tests/utils/grok"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDockerUsernamePasswordExecute(t *testing.T) {
	m := NewDockerUsernamePassword("registry", "username", "password")
	runner := new(test_domain.MockRunner)
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
	err := m.Execute(runner)
	assert.NotNil(t, err)
	assert.Equal(t, "some error", err.Error())
}

type dockerBuildTaskExecute struct {
	branch               string
	switchBranchErr      error
	dockerfile           string
	contextDir           string
	buildArgs            []string
	push                 bool
	registry             string
	imageName            string
	expectedResolves     []*test_domain.ArgumentResolution
	expectedCommands     []test_domain.CommandsExpectation
	expectedCreationErr  string
	expectedDependencies []domain.ImageDependencies
	expectedExecutionErr string
}

func testDockerBuildTaskExecute(t *testing.T, tc dockerBuildTaskExecute) {
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	runner.PrepareResolve(tc.expectedResolves)
	source := new(test_domain.MockSource)
	if tc.branch != "" {
		source.On("EnsureBranch", runner, tc.branch).Return(tc.switchBranchErr).Times(1)
	}
	target, err := NewDockerBuildTarget(source, tc.branch, tc.dockerfile, tc.contextDir, tc.buildArgs, tc.push, tc.registry, tc.imageName)
	if tc.expectedCreationErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedCreationErr), err.Error())
		return
	}
	dependencies, err := target.Build.Execute(runner)
	if tc.expectedExecutionErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedExecutionErr), err.Error())
	} else {
		assert.Nil(t, err)
		testutils.AssertSameDependencies(t, tc.expectedDependencies, dependencies)
	}
}

func TestDockerBuildTaskExecuteNoImage(t *testing.T) {
	testDockerBuildTaskExecute(t, dockerBuildTaskExecute{
		push:                true,
		expectedCreationErr: fmt.Sprintf("^When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage),
	})
}

func TestDockerBuildTaskExecuteBranchError(t *testing.T) {
	testDockerBuildTaskExecute(t, dockerBuildTaskExecute{
		branch: "bad_branch",
		expectedResolves: []*test_domain.ArgumentResolution{
			test_domain.ResolveDefault("bad_branch"),
		},
		switchBranchErr:      fmt.Errorf("bad branch"),
		expectedExecutionErr: "^Error while switching to branch bad_branch, error: bad branch$",
	})
}

func TestDockerBuildTaskExecuteMinimalParametersFailedEventually(t *testing.T) {
	// minimal (zero) parameters
	// dockerfile is not there, dependencies resolution will fail but build will go on
	// let's return a dependency error for fun
	testDockerBuildTaskExecute(t, dockerBuildTaskExecute{
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

func TestDockerBuildTaskExecuteHappy(t *testing.T) {
	// minimal (zero) parameters
	// dockerfile is not there, dependencies resolution will fail but build will go on
	// let's return a dependency error for fun
	testDockerBuildTaskExecute(t, dockerBuildTaskExecute{
		dockerfile: path.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile"),
		contextDir: "contextDir",
		buildArgs:  []string{"k1=v1", "k2=v2"},
		registry:   testutils.DotnetExampleTargetRegistryName,
		imageName:  testutils.DotnetExampleTargetImageName,
		expectedResolves: []*test_domain.ArgumentResolution{
			test_domain.ResolveDefault("../../tests/resources/docker-dotnet/Dockerfile"),
			test_domain.ResolveDefault("registry/img"),
		},
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker",
				Args:    []string{"build", "-f", "../../tests/resources/docker-dotnet/Dockerfile", "-t", testutils.DotnetExampleDependencies.Image, "--build-arg", "k1=v1", "--build-arg", "k2=v2", "contextDir"},
			},
		},
		expectedDependencies: []domain.ImageDependencies{testutils.DotnetExampleDependencies},
	})
}

func TestExport(t *testing.T) {
	task, err := NewDockerBuildTarget(nil, "myBranch", "myDockerfile", "myContextDir", []string{}, false, "myRegistry", "myImage")
	assert.Nil(t, err)
	exports := task.Export()
	assert.Equal(t, 4, len(exports))
	for _, entry := range exports {
		switch entry.Name {
		case constants.DockerfilePathVar:
			assert.Equal(t, "myDockerfile", entry.Value)
		case constants.DockerBuildContextVar:
			assert.Equal(t, "myContextDir", entry.Value)
		case constants.GitBranchVar:
			assert.Equal(t, "myBranch", entry.Value)
		case constants.DockerPushImageVar:
			assert.Equal(t, "myRegistry/myImage", entry.Value)
		default:
			assert.Fail(t, "unexpected entry: %s: %s", entry.Name, entry.Value)
		}
	}
}

type dockerPushExecuteTestCase struct {
	pushTo           string
	expectedCommands []test_domain.CommandsExpectation
	expectedErr      string
}

func testDockerPushExecute(t *testing.T, tc dockerPushExecuteTestCase) {
	task := dockerPushTask{pushTo: tc.pushTo}
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	err := task.Execute(runner)
	runner.AssertExpectations(t)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerPushExecuteNoPushTarget(t *testing.T) {
	testDockerPushExecute(t, dockerPushExecuteTestCase{
		expectedErr: "^No push target is defined$",
	})
}
func TestDockerPushExecuteHappy(t *testing.T) {
	testDockerPushExecute(t, dockerPushExecuteTestCase{
		pushTo: "someTarget",
		expectedCommands: []test_domain.CommandsExpectation{{
			Command: "docker",
			Args:    []string{"push", "someTarget"},
		}},
	})
}
