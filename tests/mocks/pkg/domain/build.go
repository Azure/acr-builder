package domain

import (
	pkg_domain "github.com/Azure/acr-builder/pkg/domain"
	"github.com/stretchr/testify/mock"
)

var _ = (pkg_domain.BuildTarget)((*MockBuildTarget)(nil))

type MockBuildTarget struct {
	mock.Mock
}

func (m *MockBuildTarget) Build(runner pkg_domain.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildTarget) Push(runner pkg_domain.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildTarget) ScanForDependencies(runner pkg_domain.Runner) ([]pkg_domain.ImageDependencies, error) {
	values := m.Called(runner)
	return values.Get(0).([]pkg_domain.ImageDependencies), values.Error(1)
}

func (m *MockBuildTarget) Export() []pkg_domain.EnvVar {
	values := m.Called()
	return values.Get(0).([]pkg_domain.EnvVar)
}

var _ = (pkg_domain.BuildSource)((*MockBuildSource)(nil))

type MockBuildSource struct {
	mock.Mock
}

func (m *MockBuildSource) Obtain(runner pkg_domain.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildSource) Return(runner pkg_domain.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}

func (m *MockBuildSource) Export() []pkg_domain.EnvVar {
	values := m.Called()
	return values.Get(0).([]pkg_domain.EnvVar)
}

var _ = (pkg_domain.DockerCredential)((*MockDockerCredential)(nil))

type MockDockerCredential struct {
	mock.Mock
}

func (m *MockDockerCredential) Authenticate(runner pkg_domain.Runner) error {
	values := m.Called(runner)
	return values.Error(0)
}
