package commands

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

type composeTestCase struct {
	path             string
	projectDir       string
	buildArgs        []string
	buildSecretArgs  []string
	expectedErr      string
	expectedCommands []test.CommandsExpectation
}

func testDockerComposeBuild(t *testing.T, tc composeTestCase) {
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	task := NewDockerComposeBuild(tc.path, tc.projectDir, tc.buildArgs, tc.buildSecretArgs)
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
		path:       filepath.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: true,
				Args:         []string{"-f", filepath.Join("docker-compose", "docker-compose.yml"), "build", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
			},
		},
	})
}

func TestDockerComposeBuildAllArgsError(t *testing.T) {
	testDockerComposeBuild(t, composeTestCase{
		path:       filepath.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: true,
				Args:         []string{"-f", filepath.Join("docker-compose", "docker-compose.yml"), "build", "--project-directory", "SomeProject", "--build-arg", "arg1=value1", "--build-arg", "arg2=value2"},
				ErrorMsg:     "Build failed",
			},
		},
		expectedErr: "^Build failed$",
	})
}

func TestDockerComposeBuildNoArgs(t *testing.T) {
	testDockerComposeBuild(t, composeTestCase{
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: true,
				Args:         []string{"build"},
			},
		},
	})
}

func testDockerComposePush(t *testing.T, tc composeTestCase) {
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	task := NewDockerComposeBuild(tc.path, tc.projectDir, tc.buildArgs, tc.buildSecretArgs)
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
		path:       filepath.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: false,
				Args:         []string{"-f", filepath.Join("docker-compose", "docker-compose.yml"), "push"},
			},
		},
	})
}

func TestDockerComposePushWithPathFailed(t *testing.T) {
	testDockerComposePush(t, composeTestCase{
		path:       filepath.Join("docker-compose", "docker-compose.yml"),
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: false,
				Args:         []string{"-f", filepath.Join("docker-compose", "docker-compose.yml"), "push"},
				ErrorMsg:     "Publish failed",
			},
		},
		expectedErr: "^Publish failed$",
	})
}

func TestDockerComposePushWithNoPath(t *testing.T) {
	testDockerComposePush(t, composeTestCase{
		buildArgs:  []string{"arg1=value1", "arg2=value2"},
		projectDir: "SomeProject",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "docker-compose",
				IsObfuscated: false,
				Args:         []string{"push"},
			},
		},
	})
}

func TestComposeBuildTaskExport(t *testing.T) {
	exports := NewDockerComposeBuild("path", "project", []string{}, []string{}).Export()
	assert.Equal(t, []build.EnvVar{
		{
			Name:  constants.ExportsDockerComposeFile,
			Value: "path",
		},
	}, exports)
}

type composeScanForDependenciesRealFileTestCase struct {
	path                 string
	expectedErr          string
	expectedDependencies []build.ImageDependencies
}

func TestComposeScanDependenciesHappy(t *testing.T) {
	testComposeScanDependenciesRealFiles(t, composeScanForDependenciesRealFileTestCase{
		path: filepath.Join("${project_root}", "docker-compose.yml"),
		expectedDependencies: []build.ImageDependencies{
			testCommon.HelloNodeExampleDependencies,
			testCommon.MultistageExampleDependencies,
		},
	})
}

func TestComposeScanDependenciesFailed(t *testing.T) {
	testComposeScanDependenciesRealFiles(t, composeScanForDependenciesRealFileTestCase{
		path: filepath.Join("${project_root}", "docker-compose-invalid.yaml"),
		expectedErr: strings.Replace(fmt.Sprintf("^Error opening docker-compose file %s",
			filepath.Join("..", "..", "tests", "resources", "docker-compose", "docker-compose-invalid.yaml")), "\\", "\\\\", -1),
	})
}

func testComposeScanDependenciesRealFiles(t *testing.T, tc composeScanForDependenciesRealFileTestCase) {
	runner := test.NewMockRunner()
	defer runner.AssertExpectations(t)
	runner.UseDefaultFileSystem()
	runner.SetContext(build.NewContext(
		append(testCommon.MultiStageExampleTestEnv,
			build.EnvVar{Name: "project_root", Value: filepath.Join("..", "..", "tests", "resources", "docker-compose")}),
		[]build.EnvVar{}))
	task := NewDockerComposeBuild(tc.path, "", []string{}, []string{})
	dep, err := task.ScanForDependencies(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
		testCommon.AssertSameDependencies(t, tc.expectedDependencies, dep)
	}
}

type composeFileProbeTestCase struct {
	files       test.FileSystemExpectations
	expectedErr string
}

func TestComposeFileProbeFailed(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", false, nil),
		expectedErr: "^No default docker-compose file found$",
	})
}

func TestComposeFileProbeSucceed(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", true, nil),
		expectedErr: "^Error opening docker-compose file docker-compose.yaml",
	})
}

func TestComposeFileProbeFSError(t *testing.T) {
	testComposeFileProbe(t, composeFileProbeTestCase{
		files: make(test.FileSystemExpectations, 0).
			AssertFileExists("docker-compose.yml", false, nil).
			AssertFileExists("docker-compose.yaml", true, fmt.Errorf("boom")),
		expectedErr: "^Unexpected error while checking for default docker compose file: boom$",
	})
}

func testComposeFileProbe(t *testing.T, tc composeFileProbeTestCase) {
	runner := test.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test.MockFileSystem)
	fs.PrepareFileSystem(tc.files)
	defer fs.AssertExpectations(t)
	_, err := NewDockerComposeBuild("", "", []string{}, []string{}).ScanForDependencies(runner)
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
}
