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
)

func TestDockerComposeBuildExecute(t *testing.T) {
	testCase := []struct {
		branch               string
		path                 string
		projectDir           string
		buildArgs            []string
		branchErr            string
		files                []*test_domain.FileSystemExpectation
		expectedErr          string
		expectedCommands     []test_domain.CommandsExpectation
		expectedDependencies []domain.ImageDependencies
	}{
		// happy path
		{
			branch:     "branch",
			path:       path.Join("docker-compose", "docker-compose.yml"),
			buildArgs:  []string{"arg1=value1", "arg2=value2"},
			projectDir: "SomeProject",
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command: "docker-compose",
					Args:    []string{"-f", "docker-compose/docker-compose.yml", "build", "--pull", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
				},
			},
		},
		// failed to switch branch
		{
			branch:      "must_switch",
			branchErr:   "Can't switch branch for some reasons....",
			expectedErr: "^Error while switching to branch must_switch, error: Can't switch branch for some reasons....",
		},
		// failed to find docker file
		{
			files: []*test_domain.FileSystemExpectation{
				test_domain.AssertFileExists("docker-compose.yml", false, nil),
				test_domain.AssertFileExists("docker-compose.yaml", false, nil),
			},
			expectedErr: "^Failed to find docker compose file in source directory",
		},
		// test case: scanning for docker-compose file, found the file but cannot parse dependency
		// however build fails afterward
		//
		// unexpected error while reading dockerfile should be logged but not cause the method to exit
		// dependencies resolution will fail, but the build will go on
		{
			files: []*test_domain.FileSystemExpectation{
				test_domain.AssertFileExists("docker-compose.yml", false, fmt.Errorf("Some filesystem error")),
				test_domain.AssertFileExists("docker-compose.yaml", true, nil),
			},
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command:  "docker-compose",
					Args:     []string{"-f", "docker-compose.yaml", "build", "--pull"},
					ErrorMsg: "Some error while building",
				},
			},
			expectedErr: "^Some error while building",
		},
	}

	for index, tc := range testCase {
		t.Logf("Running scenario %d...\n", index)
		runner := new(test_domain.MockRunner)
		runner.PrepareDefaultResolves().Maybe()
		runner.PrepareCommandExpectation(tc.expectedCommands)
		if tc.files != nil {
			runner.PrepareFileSystem(tc.files)
		}
		source := new(test_domain.MockSource)
		var branchErr error
		if tc.branchErr != "" {
			branchErr = fmt.Errorf(tc.branchErr)
		}
		if tc.branch != "" {
			source.On("EnsureBranch", runner, tc.branch).Return(branchErr).Times(1)
		}
		task := NewDockerComposeBuildTarget(source, tc.branch, tc.path, tc.projectDir, tc.buildArgs)
		dependencies, err := task.Build.Execute(runner)
		runner.AssertExpectations(t)
		source.AssertExpectations(t)
		if tc.expectedErr != "" {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
		} else {
			assert.Nil(t, err)
			testutils.AssertSameDependencies(t, tc.expectedDependencies, dependencies)
		}
	}
}

func TestComposeBuildTaskExport(t *testing.T) {
	task := NewDockerComposeBuildTarget(nil, "branch", "path", "project", []string{}).Build
	runner := new(test_domain.MockRunner)
	runner.PrepareDefaultResolves().Maybe()
	exports := task.(domain.EnvExporter).Export()
	assert.Equal(t, []domain.EnvVar{
		{
			Name:  constants.DockerComposeFileVar,
			Value: "path",
		},
		{
			Name:  constants.GitBranchVar,
			Value: "branch",
		},
	},
		exports)
}

func TestDockerComposePublishExecute(t *testing.T) {
	testCase := []struct {
		path             string
		expectedCommands []test_domain.CommandsExpectation
		expectedErr      string
	}{
		// path not given
		{
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command: "docker-compose",
					Args:    []string{"push"},
				},
			},
		},
		// path given
		{
			path: "custom-compose-file.yml",
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command: "docker-compose",
					Args:    []string{"-f", "custom-compose-file.yml", "push"},
				},
			},
		},
		// error occurred
		{
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command:  "docker-compose",
					Args:     []string{"push"},
					ErrorMsg: "Some publish error",
				},
			},
			expectedErr: "^Some publish error$",
		},
	}

	for _, tc := range testCase {
		task := NewDockerComposeBuildTarget(nil, "branch", tc.path, "project", []string{}).Push
		runner := new(test_domain.MockRunner)
		runner.PrepareCommandExpectation(tc.expectedCommands)
		err := task.Execute(runner)
		if tc.expectedErr == "" {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
		}
	}
}
