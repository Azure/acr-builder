package domain

import (
	build "github.com/Azure/acr-builder/pkg"
	"github.com/stretchr/testify/mock"
)

var _ = (build.Target)((*MockBuildTarget)(nil))

type MockBuildTarget struct {
	mock.Mock
}

func (m *MockBuildTarget) Ensure(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildTarget) Build(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildTarget) Push(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildTarget) ScanForDependencies(runner build.Runner) ([]build.ImageDependencies, error) {
	values := m.Called(runner)
	return values.Get(0).([]build.ImageDependencies), values.Error(1)
}

func (m *MockBuildTarget) Export() []build.EnvVar {
	values := m.Called()
	return values.Get(0).([]build.EnvVar)
}

var _ = (build.Source)((*MockBuildSource)(nil))

type MockBuildSource struct {
	mock.Mock
}

func (m *MockBuildSource) Obtain(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildSource) Return(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildSource) Export() []build.EnvVar {
	values := m.Called()
	return values.Get(0).([]build.EnvVar)
}

var _ = (build.DockerCredential)((*MockDockerCredential)(nil))

type MockDockerCredential struct {
	mock.Mock
}

func (m *MockDockerCredential) Authenticate(runner build.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}
