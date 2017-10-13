package grok

import (
	"path/filepath"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

func TestResolveDockerfileDependencies(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile")
	runtime, buildtimes, err := ResolveDockerfileDependencies(path)
	if err != nil {
		assert.Failf(t, "Failed", "Scenario failed with unexpected error: %s", err)
	}
	testCommon.AssertSameDependencies(t, []domain.ImageDependencies{testCommon.DotnetExampleDependencies}, []domain.ImageDependencies{
		{
			Image:             testCommon.DotnetExampleDependencies.Image,
			BuildDependencies: buildtimes,
			RuntimeDependency: runtime,
		},
	})
}
