package domain

import (
	"fmt"

	pkg_domain "github.com/Azure/acr-builder/pkg/domain"
	"github.com/stretchr/testify/mock"
)

var _ = (pkg_domain.Runner)((*MockRunner)(nil))

type MockRunner struct {
	mock.Mock
	context *pkg_domain.BuilderContext
	fs      pkg_domain.FileSystem
}

func NewMockRunner() *MockRunner {
	context := pkg_domain.NewContext([]pkg_domain.EnvVar{}, []pkg_domain.EnvVar{})
	fs := new(MockFileSystem)
	fs.SetContext(context)
	result := new(MockRunner)
	result.context = context
	result.fs = fs
	return result
}

func (m *MockRunner) GetFileSystem() pkg_domain.FileSystem {
	return m.fs
}

func (m *MockRunner) SetFileSystem(fs pkg_domain.FileSystem) {
	m.fs = fs
}

func (m *MockRunner) UseDefaultFileSystem() {
	m.fs = &pkg_domain.BuildContextAwareFileSystem{}
}

func (m *MockRunner) SetContext(context *pkg_domain.BuilderContext) {
	m.context = context
	fs, isAware := m.fs.(pkg_domain.BuildContextAware)
	if isAware {
		fs.SetContext(context)
	}
}

func (m *MockRunner) GetContext() *pkg_domain.BuilderContext {
	return m.context
}

func (m *MockRunner) ExecuteCmd(cmdExe string, cmdArgs []string) error {
	values := m.Called(cmdExe, cmdArgs)
	return values.Error(0)
}

func (m *MockRunner) ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	values := m.Called(obfuscate, cmdExe, cmdArgs)
	return values.Error(0)
}

type CommandsExpectation struct {
	Command       string
	Args          []string
	SensitiveArgs map[int]bool
	Times         *int
	ErrorMsg      string
	IsOptional    bool
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
		m.On("ExecuteCmd", cmd.Command, cmd.Args).Return(returnErr).Times(times)
	}
}

var _ = (pkg_domain.BuildContextAware)((*MockFileSystem)(nil))
var _ = (pkg_domain.FileSystem)((*MockFileSystem)(nil))

type MockFileSystem struct {
	mock.Mock
	context *pkg_domain.BuilderContext
}

func (m *MockFileSystem) GetContext() *pkg_domain.BuilderContext {
	return m.context
}

func (m *MockFileSystem) SetContext(context *pkg_domain.BuilderContext) {
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

func (m *MockFileSystem) Chdir(path string) error {
	values := m.Called(m.context.Expand(path))
	return values.Error(0)
}

func (m *MockFileSystem) PrepareFileSystem(commands FileSystemExpectations) {
	for _, cmd := range commands {
		m.On(cmd.operation, cmd.path).Return(cmd.assertion, cmd.err)
	}
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

func (e FileSystemExpectations) AssertIsDirEmpty(path string, empty bool, err error) FileSystemExpectations {
	return append(e,
		FileSystemExpectation{
			operation: "DoesFileExist",
			path:      path,
			assertion: true,
			err:       nil,
		},
		FileSystemExpectation{
			operation: "IsDirEmpty",
			path:      path,
			assertion: empty,
			err:       err,
		})
}
