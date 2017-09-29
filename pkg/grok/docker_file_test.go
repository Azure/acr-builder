package grok

import (
	"testing"

	"path"

	"github.com/Azure/acr-builder/pkg/domain"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

func TestResolveDockerfileDependencies(t *testing.T) {
	path := path.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile")
	runtime, buildtimes, err := ResolveDockerfileDependencies(path)
	if err != nil {
		assert.Failf(t, "Failed", "Scenario failed with unexpected error: %s", err)
	}
	testutils.AssertSameDependencies(t, []domain.ImageDependencies{testutils.DotnetExampleDependencies}, []domain.ImageDependencies{
		domain.ImageDependencies{
			Image:             testutils.DotnetExampleDependencies.Image,
			BuildDependencies: buildtimes,
			RuntimeDependency: runtime,
		},
	})
}
