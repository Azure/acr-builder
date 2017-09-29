package commands

import (
	"fmt"
	"path"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	test_domain "github.com/Azure/acr-builder/tests/mocks/pkg/domain"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
)

type composeTestCase struct {
	path             string
	projectDir       string
	buildArgs        []string
	expectedErr      string
	expectedCommands []test_domain.CommandsExpectation
}

func testDockerComposeBuild(t *testing.T, tc composeTestCase) {
	runner := test_domain.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	task := NewDockerComposeBuild(tc.path, tc.projectDir, tc.buildArgs)
	err := task.Build(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerComposeBuildAllArgs(t *testing.T) {
	testDockerComposeBuild(t, composeTestCase{
		path:       path.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"-f", "docker-compose/docker-compose.yml", "build", "--pull", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
			},
		},
	})
}

func TestDockerComposeBuildAllArgsError(t *testing.T) {
	testDockerComposeBuild(t, composeTestCase{
		path:       path.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command:  "docker-compose",
				Args:     []string{"-f", "docker-compose/docker-compose.yml", "build", "--pull", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
				ErrorMsg: "Build failed",
			},
		},
		expectedErr: "^Build failed$",
	})
}

func TestDockerComposeBuildNoArgs(t *testing.T) {
	testDockerComposeBuild(t, composeTestCase{
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"build", "--pull"},
			},
		},
	})
}

func testDockerComposePush(t *testing.T, tc composeTestCase) {
	runner := test_domain.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	task := NewDockerComposeBuild(tc.path, tc.projectDir, tc.buildArgs)
	err := task.Push(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
	}
}

func TestDockerComposePushWithPath(t *testing.T) {
	testDockerComposePush(t, composeTestCase{
		path:       path.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"-f", "docker-compose/docker-compose.yml", "push"},
			},
		},
	})
}

func TestDockerComposePushWithPathFailed(t *testing.T) {
	testDockerComposePush(t, composeTestCase{
		path:       path.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command:  "docker-compose",
				Args:     []string{"-f", "docker-compose/docker-compose.yml", "push"},
				ErrorMsg: "Publish failed",
			},
		},
		expectedErr: "^Publish failed$",
	})
}

func TestDockerComposePushWithNoPath(t *testing.T) {
	testDockerComposePush(t, composeTestCase{
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test_domain.CommandsExpectation{
			{
				Command: "docker-compose",
				Args:    []string{"push"},
			},
		},
	})
}

func TestComposeBuildTaskExport(t *testing.T) {
	exports := NewDockerComposeBuild("path", "project", []string{}).Export()
	assert.Equal(t, []domain.EnvVar{
		{
			Name:  constants.DockerComposeFileVar,
			Value: "path",
		},
	}, exports)
}

type composeScanForDependenciesRealFileTestCase struct {
	path                 string
	expectedErr          string
	expectedDependencies []domain.ImageDependencies
}

func TestComposeScanDependenciesHappy(t *testing.T) {
	testComposeScanDependenciesRealFiles(t, composeScanForDependenciesRealFileTestCase{
		path:                 path.Join("${project_root}", "docker-compose.yml"),
		expectedDependencies: []domain.ImageDependencies{testutils.HelloNodeExampleDependencies, testutils.MultistageExampleDependencies},
	})
}

func TestComposeScanDependenciesFailed(t *testing.T) {
	testComposeScanDependenciesRealFiles(t, composeScanForDependenciesRealFileTestCase{
		path:        path.Join("${project_root}", "docker-compose.ymll"),
		expectedErr: "no such file or directory",
	})
}

func testComposeScanDependenciesRealFiles(t *testing.T, tc composeScanForDependenciesRealFileTestCase) {
	runner := test_domain.NewMockRunner()
	defer runner.AssertExpectations(t)
	runner.UseDefaultFileSystem()
	runner.SetContext(domain.NewContext(
		append(testutils.MultiStageExampleTestEnv,
			domain.EnvVar{Name: "project_root", Value: path.Join("..", "..", "tests", "resources", "docker-compose")}),
		[]domain.EnvVar{}))
	task := NewDockerComposeBuild(tc.path, "", []string{})
	dep, err := task.ScanForDependencies(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
		testutils.AssertSameDependencies(t, tc.expectedDependencies, dep)
	}
}

type composeFileProbeTestCase struct {
	files       test_domain.FileSystemExpectations
	expectedErr string
}

func TestComposeFileProbeFailed(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test_domain.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", false, nil),
		expectedErr: "^No default docker-compose file found$",
	})
}

func TestComposeFileProbeSucceed(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test_domain.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", true, nil),
		expectedErr: "^Error opening docker-compose file docker-compose.yaml",
	})
}

func TestComposeFileProbeFSError(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test_domain.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", true, fmt.Errorf("boom")),
		expectedErr: "^Unexpected error while checking for default docker compose file: boom$",
	})
}

func testComposeFileProbe(t *testing.T, tc composeFileProbeTestCase) {
	runner := test_domain.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test_domain.MockFileSystem)
	fs.PrepareFileSystem(tc.files)
	defer fs.AssertExpectations(t)
	_, err := NewDockerComposeBuild("", "", []string{}).ScanForDependencies(runner)
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
}
