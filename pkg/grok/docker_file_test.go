package grok

import (
	"path/filepath"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

func TestResolveDockerfileDependencies(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "resources", "docker-dotnet", "Dockerfile")
	buildArgs := append(testCommon.DotnetExampleMinimalBuildArg, "build_version=2.0.5")
	runtime, buildtimes, err := ResolveDockerfileDependencies(buildArgs, path)
	if err != nil {
		assert.Failf(t, "Failed", "Scenario failed with unexpected error: %s", err)
	}
	testCommon.AssertSameDependencies(t, []build.ImageDependencies{*testCommon.NewImageDependencies(
		testCommon.DotnetExampleFullImageName,
		"microsoft/aspnetcore:2.0.6",
		[]string{"microsoft/aspnetcore-build:2.0.5", "imaginary/cert-generator:1.0"},
	)}, []build.ImageDependencies{
		*testCommon.NewImageDependencies(testCommon.DotnetExampleFullImageName, runtime, buildtimes),
	})
}
