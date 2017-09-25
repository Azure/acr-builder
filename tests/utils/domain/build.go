package domain

import (
	"fmt"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/stretchr/testify/mock"
)

type CommandsExpectation struct {
	Command       string
	Args          []string
	SensitiveArgs map[int]bool
	Times         *int
	ErrorMsg      string
	IsOptional    bool
}

type FileSystemExpectations []FileSystemExpectation

func NewFileSystem() FileSystemExpectations {
	return []FileSystemExpectation{}
}

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

type ArgumentResolution struct {
	Input  string
	Output string
}

func ResolveDefault(input string) *ArgumentResolution {
	return Resolve(input, input)
}

func Resolve(input string, output string) *ArgumentResolution {
	return &ArgumentResolution{Input: input, Output: output}
}

type MockRunner struct {
	mock.Mock
}

func (m *MockRunner) PrepareResolve(parameters []*ArgumentResolution) {
	for _, entry := range parameters {
		m.On("Resolve", entry.Input).Return(entry.Output)
	}
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

func (m *MockRunner) PrepareFileSystem(commands FileSystemExpectations) {
	for _, cmd := range commands {
		m.On(cmd.operation, cmd.path).Return(cmd.assertion, cmd.err)
	}
}

func (m *MockRunner) AppendContext(newEnv []domain.EnvVar) domain.Runner {
	values := m.Called(newEnv)
	return values.Get(0).(domain.Runner)
}

func (m *MockRunner) Resolve(value string) string {
	values := m.Called(value)
	return values.String(0)
}

func (m *MockRunner) ExecuteCmd(cmdExe string, cmdArgs []string) error {
	values := m.Called(cmdExe, cmdArgs)
	return values.Error(0)
}

func (m *MockRunner) ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	values := m.Called(obfuscate, cmdExe, cmdArgs)
	return values.Error(0)
}

func (m *MockRunner) DoesDirExist(path string) (bool, error) {
	values := m.Called(path)
	return values.Bool(0), values.Error(1)
}

func (m *MockRunner) DoesFileExist(path string) (bool, error) {
	values := m.Called(path)
	return values.Bool(0), values.Error(1)
}

func (m *MockRunner) IsDirEmpty(path string) (bool, error) {
	values := m.Called(path)
	return values.Bool(0), values.Error(1)
}

func (m *MockRunner) Chdir(path string) error {
	values := m.Called(path)
	return values.Error(0)
}

type MockSource struct {
	mock.Mock
}

func (s *MockSource) EnsureSource(runner domain.Runner) error {
	values := s.Called(runner)
	return values.Error(0)
}

func (s *MockSource) EnsureBranch(runner domain.Runner, branch string) error {
	runner.Resolve(branch)
	values := s.Called(runner, branch)
	return values.Error(0)
}

func (s *MockSource) Export() []domain.EnvVar {
	values := s.Called()
	return values.Get(0).([]domain.EnvVar)
}
