package domain

import (
	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/shhsu/testify/mock"
)

type CommandsExpectation struct {
	Command       string
	Args          []string
	SensitiveArgs map[int]bool
	Times         *int
	ErrorMsg      string
	IsOptional    bool
}

type FileSystemExpectation struct {
	operation string
	path      string
	assertion bool
	err       error
}

func AssertFileExists(path string, exists bool, err error) *FileSystemExpectation {
	return &FileSystemExpectation{
		operation: "DoesFileExist",
		path:      path,
		assertion: exists,
		err:       err,
	}
}

func AssertDirExists(path string, exists bool, err error) *FileSystemExpectation {
	return &FileSystemExpectation{
		operation: "DoesDirExist",
		path:      path,
		assertion: exists,
		err:       err,
	}
}

func AssertIsDirEmpty(path string, empty bool, err error) *FileSystemExpectation {
	return &FileSystemExpectation{
		operation: "IsDirEmpty",
		path:      path,
		assertion: empty,
		err:       err,
	}
}

type MockRunner struct {
	mock.Mock
}

func (m *MockRunner) PrepareDefaultResolves() *mock.Call {
	resolveCall := m.On("Resolve", mock.Anything)
	return resolveCall.Run(func(arg mock.Arguments) {
		input := arg.Get(0).(string)
		resolveCall.ReturnArguments = []interface{}{input}
	})
}

func (m *MockRunner) PrepareEnvResolves(env map[string]string) *mock.Call {
	resolveCall := m.On("Resolve", mock.Anything)
	return resolveCall.Run(func(arg mock.Arguments) {
		input := arg.Get(0).(string)
		value := os.Expand(input, func(k string) string {
			v, _ := env[k]
			return v
		})
		resolveCall.ReturnArguments = []interface{}{value}
	})
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
		call := m.On("ExecuteCmd", cmd.Command, cmd.Args).Return(returnErr).Times(times)
		if cmd.IsOptional {
			call.Maybe()
		}
	}
}

func (m *MockRunner) PrepareFileSystem(commands []*FileSystemExpectation) {
	for _, cmd := range commands {
		m.On(cmd.operation, cmd.path).Return(cmd.assertion, cmd.err).Maybe()
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

func (m *MockRunner) ExecuteString(cmdString string) error {
	values := m.Called(cmdString)
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
