package domain

import (
	"fmt"
	"io"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/mock"
)

var _ = (build.Runner)((*MockRunner)(nil))

type MockRunner struct {
	mock.Mock
	context *build.BuilderContext
	fs      build.FileSystem
	reader  io.ReadCloser
}

func NewMockRunner() *MockRunner {
	context := build.NewContext([]build.EnvVar{}, []build.EnvVar{})
	fs := new(MockFileSystem)
	fs.SetContext(context)
	result := new(MockRunner)
	result.context = context
	result.fs = fs
	return result
}

func (m *MockRunner) CreateStdinPipeWriter() (writer io.WriteCloser) {
	m.reader, writer = io.Pipe()
	return
}

func (m *MockRunner) GetStdin() io.Reader {
	return m.reader
}

func (m *MockRunner) GetFileSystem() build.FileSystem {
	return m.fs
}

func (m *MockRunner) SetFileSystem(fs build.FileSystem) {
	m.fs = fs
}

func (m *MockRunner) UseDefaultFileSystem() {
	m.fs = build.NewContextAwareFileSystem(m.context)
}

func (m *MockRunner) SetContext(context *build.BuilderContext) {
	m.context = context
	fs, isAware := m.fs.(build.ContextAware)
	if isAware {
		fs.SetContext(context)
	}
}

func (m *MockRunner) GetContext() *build.BuilderContext {
	return m.context
}

func (m *MockRunner) ExecuteCmd(cmdExe string, cmdArgs []string, reader io.Reader) error {
	values := m.Called(cmdExe, cmdArgs, reader)
	return values.Error(0)
}

func (m *MockRunner) QueryCmd(cmdExe string, cmdArgs []string) (string, error) {
	values := m.Called(cmdExe, cmdArgs)
	return values.String(0), values.Error(1)
}

func (m *MockRunner) ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	values := m.Called(obfuscate, cmdExe, cmdArgs)
	return values.Error(0)
}

func (m *MockRunner) PrepareDigestQuery(
	expectedDependencies []build.ImageDependencies,
	queryCmdErr map[string]error) {
	for _, expectedDep := range expectedDependencies {
		if expectedDep.Image != nil {
			m.addQuery(expectedDep.Image, queryCmdErr[expectedDep.Image.String()])
		}
		m.addQuery(expectedDep.Runtime, queryCmdErr[expectedDep.Runtime.String()])
		for _, expectedBuildtime := range expectedDep.Buildtime {
			m.addQuery(expectedBuildtime, queryCmdErr[expectedBuildtime.String()])
		}
	}
}

func (m *MockRunner) PrepareGitSHAQuery(value string, err error) {
	m.On("QueryCmd", "git", []string{"rev-parse", "--verify", "HEAD"}).Return(value, err).Once()
}

func (m *MockRunner) addQuery(reference *build.ImageReference, err error) {
	refKey := reference.String()
	var result string
	if err == nil {
		result = testCommon.GetRepoDigests(refKey)
	}
	m.On("QueryCmd", "docker", []string{"inspect", "--format", "\"{{json .RepoDigests}}\"", refKey}).Return(result, err)
}

func (m *MockRunner) Close() error {
	if m.reader != nil {
		return m.reader.Close()
	}
	return nil
}

type CommandsExpectation struct {
	Command       string
	Args          []string
	SensitiveArgs map[int]bool
	Times         *int
	ErrorMsg      string
	IsObfuscated  bool
	QueryResult   string
}

func (m *MockRunner) PrepareCommandExpectation(commands []CommandsExpectation) {
	for _, cmd := range commands {
		times := 1
		if cmd.Times != nil {
			times = *cmd.Times
		}
		returnErr := error(nil)
		if cmd.ErrorMsg != "" {
			returnErr = fmt.Errorf(cmd.ErrorMsg)
		}
		if len(cmd.QueryResult) > 0 {
			m.On("QueryCmd", cmd.Command, cmd.Args).Return(cmd.QueryResult, returnErr).Times(times)
		} else if cmd.IsObfuscated {
			m.On("ExecuteCmdWithObfuscation", mock.Anything, cmd.Command, cmd.Args).Return(returnErr).Times(times)
		} else {
			m.On("ExecuteCmd", cmd.Command, cmd.Args, mock.Anything).Return(returnErr).Times(times)
		}
	}
}

var _ = (build.ContextAware)((*MockFileSystem)(nil))
var _ = (build.FileSystem)((*MockFileSystem)(nil))

type MockFileSystem struct {
	mock.Mock
	context *build.BuilderContext
}

func (m *MockFileSystem) GetContext() *build.BuilderContext {
	return m.context
}

func (m *MockFileSystem) SetContext(context *build.BuilderContext) {
	m.context = context
}

func (m *MockFileSystem) DoesDirExist(path string) (bool, error) {
	values := m.Called(m.context.Expand(path))
	return values.Bool(0), values.Error(1)
}

func (m *MockFileSystem) DoesFileExist(path string) (bool, error) {
	values := m.Called(m.context.Expand(path))
	return values.Bool(0), values.Error(1)
}

func (m *MockFileSystem) IsDirEmpty(path string) (bool, error) {
	values := m.Called(m.context.Expand(path))
	return values.Bool(0), values.Error(1)
}

func (m *MockFileSystem) Getwd() (string, error) {
	values := m.Called()
	return values.String(0), values.Error(1)
}

func (m *MockFileSystem) Chdir(path string) error {
	values := m.Called(m.context.Expand(path))
	return values.Error(0)
}

func (m *MockFileSystem) PrepareFileSystem(commands FileSystemExpectations) {
	for _, cmd := range commands {
		m.On(cmd.operation, cmd.path).Return(cmd.assertion, cmd.err).Once()
	}
}

func (m *MockFileSystem) PrepareChdir(expectations ChdirExpectations) {
	for _, exp := range expectations {
		m.On("Chdir", exp.Path).Return(exp.Err).Once()
	}
}

func (m *MockFileSystem) WriteFile(path string, source io.Reader) error {
	values := m.Called(m.context.Expand(path), source)
	return values.Error(0)
}

func (m *MockFileSystem) CreateTempDir() (string, error) {
	values := m.Called()
	return values.String(0), values.Error(1)
}

func (m *MockFileSystem) Cleanup() {
	m.Called()
}

type FileSystemExpectations []FileSystemExpectation

type FileSystemExpectation struct {
	operation string
	path      string
	assertion bool
	err       error
}

func (e FileSystemExpectations) AssertFileExists(path string, exists bool, err error) FileSystemExpectations {
	return append(e, FileSystemExpectation{
		operation: "DoesFileExist",
		path:      path,
		assertion: exists,
		err:       err,
	})
}

func (e FileSystemExpectations) AssertDirExists(path string, exists bool, err error) FileSystemExpectations {
	return append(e, FileSystemExpectation{
		operation: "DoesDirExist",
		path:      path,
		assertion: exists,
		err:       err,
	})
}

func (e FileSystemExpectations) AssertPwdEmpty(empty bool, err error) FileSystemExpectations {
	return append(e,
		FileSystemExpectation{
			operation: "IsDirEmpty",
			path:      ".",
			assertion: empty,
			err:       err,
		})
}

func (e FileSystemExpectations) AssertIsDirEmpty(path string, empty bool, err error) FileSystemExpectations {
	if path != "." {
		e = append(e, FileSystemExpectation{
			operation: "DoesDirExist",
			path:      path,
			assertion: true,
			err:       nil,
		})
	}
	return append(e,
		FileSystemExpectation{
			operation: "IsDirEmpty",
			path:      path,
			assertion: empty,
			err:       err,
		})
}

type ChdirExpectations []ChdirExpectation

type ChdirExpectation struct {
	Path string
	Err  error
}
