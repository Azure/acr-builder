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

type passThroughSourceTestCase struct {
	workingDir           string
	context              string
	stdinWriteFunc       func(io.Writer)
	dockerfile           string
	tempDirCreated       bool
	expectedNewSourceErr string
	expectedObtainErr    string
	expectedFiles        []string
	expectedFileContent  map[string][]byte
}

func testPassThroughSourceObtain(t *testing.T, tc passThroughSourceTestCase) {
	source, err := NewPassThroughSource(tc.workingDir, tc.context, tc.dockerfile)
	testCommon.AssertErrorPattern(t, tc.expectedNewSourceErr, err)
	runner := test.NewMockRunner()
	fs := build.NewContextAwareFileSystem(runner.GetContext())
	defer fs.Cleanup()
	runner.SetFileSystem(fs)
	if tc.stdinWriteFunc != nil {
		writer := runner.CreateStdinPipeWriter()
		go func() {
			defer writer.Close()
			tc.stdinWriteFunc(writer)
		}()
	}
	defer testCommon.ReportOnError(t, runner.Close)
	err = source.Obtain(runner)
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
	}
}

func TestObtainGitSuccess(t *testing.T) {
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
		context:       "https://github.com/Azure/acr-builder.git",
		expectedFiles: testCommon.ExpectedACRBuilderRelativePaths,
	})
}

func TestObtainGitBadRequest(t *testing.T) {
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
		context:           "https://github.com/Azure/must-not-exists.git",
		expectedObtainErr: "^unable to 'git clone' to temporary context directory",
	})
}

func TestStreamArchiveExists(t *testing.T) {
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
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
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
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
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
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
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
		context:       testCommon.MultiStageExampleRoot,
		expectedFiles: testCommon.ExpectedMultiStageExampleRelativePaths,
	})
}

func TestLocalDirIsFile(t *testing.T) {
	path := filepath.Join(testCommon.MultiStageExampleRoot, "Dockerfile")
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
		context:           path,
		expectedObtainErr: fmt.Sprintf("^Failed to look up context from path %s", strings.Replace(path, "\\", "\\\\", -1)),
	})
}

func TestLocalDirDNE(t *testing.T) {
	path := filepath.Join(testCommon.Config.ProjectRoot, "must-not-exists")
	testPassThroughSourceObtain(t, passThroughSourceTestCase{
		context:           path,
		expectedObtainErr: fmt.Sprintf("^Unable to determine context type for context \"%s\".", strings.Replace(path, "\\", "\\\\", -1)),
	})
}

type webPassThroughSourceTestCase struct {
	passThroughSourceTestCase
	handler http.Handler
}

func testWebPassThroughSourceObtain(t *testing.T, tc webPassThroughSourceTestCase) {
	server := testCommon.StartStaticFileServer(t, tc.handler)
	defer testCommon.ReportOnError(t, func() error { return server.Shutdown(context.TODO()) })
	testPassThroughSourceObtain(t, tc.passThroughSourceTestCase)
}

func TestWebArchiveExists(t *testing.T) {
	testWebPassThroughSourceObtain(t, webPassThroughSourceTestCase{
		passThroughSourceTestCase: passThroughSourceTestCase{
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
	testWebPassThroughSourceObtain(t, webPassThroughSourceTestCase{
		passThroughSourceTestCase: passThroughSourceTestCase{
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
	testWebPassThroughSourceObtain(t, webPassThroughSourceTestCase{
		passThroughSourceTestCase: passThroughSourceTestCase{
			context:             testCommon.StaticFileHost,
			tempDirCreated:      true,
			expectedFileContent: map[string][]byte{"Dockerfile": content},
		},
		handler: &testCommon.FixedResponseHandler{
			Body: content,
		},
	})
}
