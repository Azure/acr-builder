package commands

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

type DockerSourceTestCase struct {
	context             string
	stdinWriteFunc      func(io.Writer)
	gitSha              string
	gitShaError         error
	dockerfile          string
	tempDirCreated      bool
	expectedObtainErr   string
	expectedFiles       []string
	expectedFileContent map[string][]byte
}

func testDockerSourceObtain(t *testing.T, tc DockerSourceTestCase) {
	source := NewDockerSource(tc.context, tc.dockerfile)
	runner := test.NewMockRunner()
	fs := build.NewContextAwareFileSystem(runner.GetContext())
	defer fs.Cleanup()
	runner.SetFileSystem(fs)
	if tc.stdinWriteFunc != nil {
		writer := runner.CreateStdinPipeWriter()
		go func() {
			defer testCommon.ReportOnError(t, writer.Close)
			tc.stdinWriteFunc(writer)
		}()
	}
	defer testCommon.ReportOnError(t, runner.Close)
	if tc.expectedObtainErr == "" {
		runner.PrepareGitSHAQuery(tc.gitSha, tc.gitShaError)
	}
	err := source.Obtain(runner)
	defer testCommon.ReportOnError(t, func() (err error) {
		return source.Return(runner)
	})
	testCommon.AssertErrorPattern(t, tc.expectedObtainErr, err)
	tempDirs := fs.TempDirsCreated()
	pwd, err := fs.Getwd()
	assert.Nil(t, err)
	if tc.tempDirCreated {
		assert.Equal(t, 1, len(tempDirs))
		tempDir := tempDirs[0]
		assert.Equal(t, tempDir, pwd)
	} else {
		assert.Empty(t, tempDirs)
	}
	isEmpty, err := fs.IsDirEmpty(pwd)
	assert.Nil(t, err)
	assert.False(t, isEmpty)
	for _, file := range tc.expectedFiles {
		exists, err := fs.DoesFileExist(file)
		assert.Nil(t, err)
		assert.True(t, exists)
	}
	for name, content := range tc.expectedFileContent {
		exists, err := fs.DoesFileExist(name)
		assert.Nil(t, err)
		assert.True(t, exists)
		actualContent, err := ioutil.ReadFile(name)
		assert.Equal(t, content, actualContent)
		if err != nil {
			assert.Fail(t, err.Error())
		}
	}
}

// TODO: annoying/flaky test that requires github access
// func TestObtainGitSuccess(t *testing.T) {
// 	testDockerSourceObtain(t, DockerSourceTestCase{
// 		context:       "https://github.com/Azure/acr-builder.git",
// 		gitSha:        "ABCDE",
// 		expectedFiles: testCommon.ExpectedACRBuilderRelativePaths,
// 	})
// }

// TODO: annoying/flaky test that requires github access
// func TestObtainGitBadRequest(t *testing.T) {
// 	testDockerSourceObtain(t, DockerSourceTestCase{
// 		context:           "https://github.com/Azure/must-not-exists.git",
// 		expectedObtainErr: "^unable to 'git clone' to temporary context directory",
// 	})
// }

func TestStreamArchiveExists(t *testing.T) {
	testDockerSourceObtain(t, DockerSourceTestCase{
		context: "-",
		stdinWriteFunc: func(w io.Writer) {
			testCommon.ReportOnError(t, func() error {
				return testCommon.StreamArchiveFromDir(t, testCommon.MultiStageExampleRoot, w)
			})
		},
		tempDirCreated: true,
		expectedFiles:  testCommon.ExpectedMultiStageExampleRelativePaths,
	})
}

func TestStreamArchiveStreamDockerfile(t *testing.T) {
	content := []byte("FROM golang")
	testDockerSourceObtain(t, DockerSourceTestCase{
		context: "-",
		stdinWriteFunc: func(w io.Writer) {
			written, err := w.Write(content)
			assert.Nil(t, err)
			assert.Equal(t, len(content), written)
		},
		tempDirCreated:      true,
		expectedFileContent: map[string][]byte{"Dockerfile": content},
	})
}

func TestStreamArchiveStreamDockerfileCustomLocation(t *testing.T) {
	content := []byte("FROM golang")
	location := "Dockerfile.my"
	testDockerSourceObtain(t, DockerSourceTestCase{
		context: "-",
		stdinWriteFunc: func(w io.Writer) {
			written, err := w.Write(content)
			assert.Nil(t, err)
			assert.Equal(t, len(content), written)
		},
		dockerfile:          location,
		tempDirCreated:      true,
		expectedFileContent: map[string][]byte{location: content},
	})
}

func TestLocalDirExists(t *testing.T) {
	testDockerSourceObtain(t, DockerSourceTestCase{
		context:       testCommon.MultiStageExampleRoot,
		expectedFiles: testCommon.ExpectedMultiStageExampleRelativePaths,
	})
}

func TestLocalDirIsFile(t *testing.T) {
	path := filepath.Join(testCommon.MultiStageExampleRoot, "Dockerfile")
	testDockerSourceObtain(t, DockerSourceTestCase{
		context:           path,
		expectedObtainErr: fmt.Sprintf("^Failed to look up context from path %s", strings.Replace(path, "\\", "\\\\", -1)),
	})
}

func TestLocalDirDNE(t *testing.T) {
	path := filepath.Join(testCommon.Config.ProjectRoot, "must-not-exists")
	testDockerSourceObtain(t, DockerSourceTestCase{
		context:           path,
		expectedObtainErr: fmt.Sprintf("^Unable to determine context type for context \"%s\".", strings.Replace(path, "\\", "\\\\", -1)),
	})
}

type webDockerSourceTestCase struct {
	DockerSourceTestCase
	handler http.Handler
}

func testWebDockerSourceObtain(t *testing.T, tc webDockerSourceTestCase) {
	server := testCommon.StartStaticFileServer(t, tc.handler)
	defer testCommon.ReportOnError(t, func() error { return server.Shutdown(context.TODO()) })
	testDockerSourceObtain(t, tc.DockerSourceTestCase)
}

func TestWebArchiveExists(t *testing.T) {
	testWebDockerSourceObtain(t, webDockerSourceTestCase{
		DockerSourceTestCase: DockerSourceTestCase{
			context:        testCommon.StaticFileHost,
			tempDirCreated: true,
			expectedFiles:  testCommon.ExpectedMultiStageExampleRelativePaths,
		},
		handler: &testCommon.StaticArchiveHandler{
			T:           t,
			ArchiveRoot: testCommon.MultiStageExampleRoot,
		},
	})
}

func TestWebArchiveBadRequest(t *testing.T) {
	testWebDockerSourceObtain(t, webDockerSourceTestCase{
		DockerSourceTestCase: DockerSourceTestCase{
			context:           testCommon.StaticFileHost,
			expectedObtainErr: fmt.Sprintf("^unable to download remote context %s", testCommon.StaticFileHost),
		},
		handler: &testCommon.FixedResponseHandler{
			ErrorMessage: "Not Found",
			StatusCode:   404,
		},
	})
}

func TestWebDockerfile(t *testing.T) {
	content := []byte("FROM golang")
	testWebDockerSourceObtain(t, webDockerSourceTestCase{
		DockerSourceTestCase: DockerSourceTestCase{
			context:             testCommon.StaticFileHost,
			tempDirCreated:      true,
			expectedFileContent: map[string][]byte{"Dockerfile": content},
		},
		handler: &testCommon.FixedResponseHandler{
			Body: content,
		},
	})
}

func TestIsVstsGitURL(t *testing.T) {
	assert.True(t, isVstsGitURL("https://tenant.visualstudio.com/org/team/_git/project"))
	assert.True(t, isVstsGitURL("https://tenant.visualstudio.com/org/team/_git/project:src"))
	assert.True(t, isVstsGitURL("https://token@tenant.visualstudio.com/org/team/_git/project#master"))
	assert.True(t, isVstsGitURL("https://user:password@tenant.visualstudio.com/org/team/_git/project#master:src"))
	assert.False(t, isVstsGitURL("https://tenant.visualstudio.com/org/team/_git/project?path=src"))
	assert.False(t, isVstsGitURL(""))
	assert.False(t, isVstsGitURL("a"))
	assert.False(t, isVstsGitURL("https://"))
	assert.False(t, isVstsGitURL("https://bing.com"))
}
