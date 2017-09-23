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
	"github.com/shhsu/testify/assert"
	"github.com/shhsu/testify/mock"
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

func TestDockerBuildTaskExecute(t *testing.T) {
	testCase := []struct {
		branch               string
		switchBranchErr      error
		dockerfile           string
		contextDir           string
		buildArgs            []string
		push                 bool
		registry             string
		imageName            string
		expectedCommands     []test_domain.CommandsExpectation
		expectedCreationErr  string
		expectedDependencies []domain.ImageDependencies
		expectedExecutionErr string
	}{
		// push but no image name
		{
			push:                true,
			expectedCreationErr: fmt.Sprintf("^When building with dockerfile, docker image name --%s is required for pushing", constants.ArgNameDockerImage),
		},
		// branch switch error
		{
			branch:               "bad_branch",
			switchBranchErr:      fmt.Errorf("bad branch"),
			expectedExecutionErr: "^Error while switching to branch bad_branch, error: bad branch$",
		},
		// minimal (zero) parameters
		// dockerfile is not there, dependencies resolution will fail but build will go on
		// let's return a dependency error for fun
		{
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command:  "docker",
					Args:     []string{"build", "."},
					ErrorMsg: "docker build error",
				},
			},
			expectedExecutionErr: "^docker build error$",
		},
		// lots of parameters
		{
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
			expectedDependencies: []domain.ImageDependencies{testutils.DotnetExampleDependencies},
		},
	}

	for index, tc := range testCase {
		t.Logf("Running test case %d...\n", index)
		runner := new(test_domain.MockRunner)
		runner.PrepareCommandExpectation(tc.expectedCommands)
		runner.PrepareDefaultResolves().Maybe()
		source := new(test_domain.MockSource)
		if tc.branch != "" {
			source.On("EnsureBranch", runner, tc.branch).Return(tc.switchBranchErr).Times(1)
		}
		target, err := NewDockerBuildTarget(source, tc.branch, tc.dockerfile, tc.contextDir, tc.buildArgs, tc.push, tc.registry, tc.imageName)
		if tc.expectedCreationErr != "" {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedCreationErr), err.Error())
			continue
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
}
