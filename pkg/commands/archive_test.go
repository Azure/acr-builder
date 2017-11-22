package commands

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

type obtainTestCase struct {
	url               string
	targetDir         string
	getWdErr          *error
	expectedChdir     test.ChdirExpectations
	expectedCommands  []test.CommandsExpectation
	expectedObtainErr string
	expectedExports   []build.EnvVar
	expectedReturnErr string
}

func TestObtainFromKnownLocation(t *testing.T) {
	targetDir := filepath.Join(testCommon.Config.ProjectRoot, "tests", "workspace")
	testArchiveSource(t,
		obtainTestCase{
			url:       testCommon.StaticFileHost,
			targetDir: targetDir,
			expectedChdir: []test.ChdirExpectation{
				{Path: targetDir},
				{Path: "home"},
			},
			getWdErr: &testCommon.NilError,
			expectedExports: []build.EnvVar{
				{Name: constants.ExportsWorkingDir, Value: targetDir},
			},
		},
	)
}

func testArchiveSource(t *testing.T, tc obtainTestCase) {
	cleanup(tc.targetDir)
	defer cleanup(tc.targetDir)
	server := testCommon.StartStaticFileServer(t)
	defer testCommon.ReportOnError(t, func() error { return server.Shutdown(nil) })
	source := NewArchiveSource(tc.url, tc.targetDir)
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	fs := runner.GetFileSystem().(*test.MockFileSystem)
	fs.PrepareChdir(tc.expectedChdir)
	if tc.getWdErr != nil {
		fs.On("Getwd").Return("home", *tc.getWdErr).Once()
	}
	err := source.Obtain(runner)
	if tc.expectedObtainErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedObtainErr), err.Error())
		return
	}
	assert.Nil(t, err)

	var projectDockerComposeFile string
	if tc.targetDir == "" {
		projectDockerComposeFile = "docker-compose.yml"
	} else {
		projectDockerComposeFile = filepath.Join(tc.targetDir, "docker-compose.yml")
	}
	_, err = os.Stat(projectDockerComposeFile)
	assert.Nil(t, err)

	exports := source.Export()
	assert.Equal(t, tc.expectedExports, exports)
	err = source.Return(runner)
	if tc.expectedReturnErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedReturnErr), err.Error())
		return
	}
	assert.Nil(t, err)
}

func cleanup(targetDir string) {
	if targetDir != "" {
		err := os.RemoveAll(targetDir)
		if err != nil {
			panic("Cleanup error: " + err.Error())
		}
	}
}
