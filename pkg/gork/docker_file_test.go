package gork

import "testing"

import "path"

import "github.com/stretchr/testify/assert"

func TestResolveDockerfileDependencies(t *testing.T) {
	path := path.Join("..", "..", "test", "resources", "docker-dotnet", "Dockerfile")
	runtime, buildtimes, err := ResolveDockerfileDependencies(path)
	if err != nil {
		assert.Failf(t, "Failed", "Scenario failed with unexpected error: %s", err)
	}
	assert.Equal(t, "microsoft/aspnetcore:2.0", runtime, "Runtime image check failed")
	expectedBuildtimes := []string{"microsoft/aspnetcore-build:2.0", "imaginary/cert-generator:1.0"}
	assert.Equal(t, 2, len(buildtimes), "Incorrect number of runtime dependencies. Expected: %v, Actual, %v", expectedBuildtimes, buildtimes)
	assert.Subset(t, expectedBuildtimes, buildtimes, "Expected dependencies. Expected: %v, Actual, %v", expectedBuildtimes, buildtimes)
}
