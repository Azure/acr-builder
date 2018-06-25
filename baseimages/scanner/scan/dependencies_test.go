package scan

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestResolveDockerfileDependencies tests resolving runtime and build time dependencies from a Dockerfile.
func TestResolveDockerfileDependencies(t *testing.T) {
	ver := "2.0.6"
	buildImg := "aspnetcore-build"

	expectedRuntime := fmt.Sprintf("microsoft/aspnetcore:%s", ver)
	expectedBuildDeps := map[string]bool{
		fmt.Sprintf("microsoft/%s:%s", buildImg, ver): true,
		"imaginary/cert-generator:1.0":                true,
	}

	path := filepath.Join("testdata", "multistage-dep-dockerfile")
	args := []string{fmt.Sprintf("build_image=%s", buildImg), fmt.Sprintf("build_version=%s", ver)}

	runtimeDep, buildDeps, err := ResolveDockerfileDependencies(path, args)

	if err != nil {
		t.Errorf("Failed to resolve dependencies: %v", err)
	}

	if runtimeDep != expectedRuntime {
		t.Errorf("Unexpected runtime. Got %s, expected %s", runtimeDep, expectedRuntime)
	}

	for _, buildDep := range buildDeps {
		if ok := expectedBuildDeps[buildDep]; !ok {
			t.Errorf("Unexpected build-time dependencies. Got %v which wasn't expected", buildDep)
		}
	}

}
