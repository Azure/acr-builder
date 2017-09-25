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
)

type dockerComposeBuildExecuteTestCase struct {
	branch               string
	path                 string
	projectDir           string
	buildArgs            []string
	branchErr            string
	files                test_domain.FileSystemExpectations
	expectedErr          string
	expectedCommands     []test_domain.CommandsExpectation
	expectedDependencies []domain.ImageDependencies
	expectedResolves     []*test_domain.ArgumentResolution
}

func testDockerComposeBuildExecute(t *testing.T, tc dockerComposeBuildExecuteTestCase) {
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	runner.PrepareResolve(tc.expectedResolves)
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

func TestDockerComposeBuildExecuteHappy(t *testing.T) {
	testDockerComposeBuildExecute(t, dockerComposeBuildExecuteTestCase{
		branch:     "myBranch",
		path:       path.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"-f", "docker-compose/docker-compose.yml", "build", "--pull", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
			},
		},
		expectedResolves: []*test_domain.ArgumentResolution{
			test_domain.ResolveDefault("myBranch"),
			test_domain.ResolveDefault("SomeProject"),
			test_domain.ResolveDefault("docker-compose/docker-compose.yml"),
		}})
}

func TestDockerComposeBuildExecuteSwithBranchError(t *testing.T) {
	testDockerComposeBuildExecute(t, dockerComposeBuildExecuteTestCase{
		branch:      "myBranch",
		branchErr:   "Can't switch branch for some reasons....",
		expectedErr: "^Error while switching to branch myBranch, error: Can't switch branch for some reasons....",
		expectedResolves: []*test_domain.ArgumentResolution{
			test_domain.ResolveDefault("myBranch"),
		},
	})
}

func TestDockerComposeBuildExecuteFailedToFindDockerfile(t *testing.T) {
	testDockerComposeBuildExecute(t, dockerComposeBuildExecuteTestCase{
		files: test_domain.NewFileSystem().
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", false, nil),
		expectedErr: "^Failed to find docker compose file in source directory",
	})
}

func TestDockerComposeBuildExecuteFailedReadDockerfile(t *testing.T) {
	// test case: scanning for docker-compose file, found the file but cannot parse dependency
	// however build fails afterward
	//
	// unexpected error while reading dockerfile should be logged but not cause the method to exit
	// dependencies resolution will fail, but the build will go on
	testDockerComposeBuildExecute(t, dockerComposeBuildExecuteTestCase{
		files: test_domain.NewFileSystem().
			AssertFileExists("docker-compose.yml", false, fmt.Errorf("Some filesystem error")).
			AssertFileExists("docker-compose.yaml", true, nil),
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command:  "docker-compose",
				Args:     []string{"-f", "docker-compose.yaml", "build", "--pull"},
				ErrorMsg: "Some error while building",
			},
		},
		expectedResolves: []*test_domain.ArgumentResolution{
			test_domain.ResolveDefault(""),
			test_domain.ResolveDefault("docker-compose.yaml"),
		},
		expectedErr: "^Some error while building",
	})
}

func TestComposeBuildTaskExport(t *testing.T) {
	task := NewDockerComposeBuildTarget(nil, "branch", "path", "project", []string{}).Build
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

type dockerComposePublishExecuteTestCase struct {
	path             string
	expectedCommands []test_domain.CommandsExpectation
	expectedErr      string
}

func testDockerComposePublishExecute(t *testing.T, tc dockerComposePublishExecuteTestCase) {
	task := NewDockerComposeBuildTarget(nil, "branch", tc.path, "project", []string{}).Push
	runner := new(test_domain.MockRunner)
	runner.PrepareCommandExpectation(tc.expectedCommands)
	err := task.Execute(runner)
	runner.AssertExpectations(t)
	if tc.expectedErr == "" {
		assert.Nil(t, err)
	} else {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	}
}

func TestDockerComposePublishExecutePathNotGiven(t *testing.T) {
	testDockerComposePublishExecute(t,
		dockerComposePublishExecuteTestCase{
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command: "docker-compose",
					Args:    []string{"push"},
				},
			},
		})
}

func TestDockerComposePublishExecutePathGiven(t *testing.T) {
	testDockerComposePublishExecute(t,
		dockerComposePublishExecuteTestCase{
			path: "custom-compose-file.yml",
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command: "docker-compose",
					Args:    []string{"-f", "custom-compose-file.yml", "push"},
				},
			},
		})
}

func TestDockerComposePublishExecuteError(t *testing.T) {
	testDockerComposePublishExecute(t,
		dockerComposePublishExecuteTestCase{
			expectedCommands: []test_domain.CommandsExpectation{
				{
					Command:  "docker-compose",
					Args:     []string{"push"},
					ErrorMsg: "Some publish error",
				},
			},
			expectedErr: "^Some publish error$",
		})
}
