package builder

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Azure/acr-builder/util"
)

// TestResolveDockerfileDependencies tests resolving runtime and build time dependencies from a Dockerfile.
func TestResolveDockerfileDependencies(t *testing.T) {
	ver := "2.0.6"
	buildImg := "aspnetcore-build"

	expectedRuntime := fmt.Sprintf("microsoft/aspnetcore:%s", ver)
	expectedBuildDeps := []string{fmt.Sprintf("microsoft/%s:%s", buildImg, ver), "imaginary/cert-generator:1.0"}

	path := filepath.Join("testdata", "multistage-dep-dockerfile")
	args := []string{fmt.Sprintf("build_image=%s", buildImg), fmt.Sprintf("build_version=%s", ver)}

	runtimeDep, buildDeps, err := ResolveDockerfileDependencies(path, args)

	if err != nil {
		t.Errorf("Failed to resolve dependencies: %v", err)
	}

	if runtimeDep != expectedRuntime {
		t.Errorf("Unexpected runtime. Got %s, expected %s", runtimeDep, expectedRuntime)
	}

	if !util.StringSequenceEquals(buildDeps, expectedBuildDeps) {
		t.Errorf("Unexpected build-time dependencies. Got %v, expected %v", buildDeps, expectedBuildDeps)
	}
}
