package gork

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

type envResolver struct {
	content map[string]string
}

func (r *envResolver) GetEnv(key string) (string, bool) {
	value, found := r.content[key]
	return value, found
}

func TestParse(t *testing.T) {
	composeFile := path.Join("..", "..", "test", "resources", "docker-compose", "docker-compose-envs.yml")
	err := os.Setenv("ACR_BUILD_DOCKER_REGISTRY", "unit-tests")
	if err != nil {
		panic(err)
	}
	entries, err := ResolveDockerComposeDependencies(
		&envResolver{
			content: map[string]string{
				"DOCKERFILE": "Dockerfile",
			},
		}, "", composeFile)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	expectedDependencies := map[string]struct {
		Runtime    string
		Buildtimes []string
	}{
		"unit-tests/hello-multistage": {
			Runtime:    "alpine",
			Buildtimes: []string{"golang:alpine"},
		},

		"unit-tests/hello-node": {
			Runtime:    "node:alpine",
			Buildtimes: []string{},
		},
	}
	assert.Equal(t, len(expectedDependencies), len(entries), "Unexpected numbers of image dependencies")
	for _, entry := range entries {
		expected, found := expectedDependencies[entry.Image]
		assert.True(t, found, "Unexpected image name: %s", entry.Image)
		assert.Equal(t, expected.Runtime, entry.RuntimeDependency)
		assert.Equal(t, len(expected.Buildtimes), len(entry.BuildDependencies),
			"Incorrect number of runtime dependencies. Expected: %v, Actual, %v", expected.Buildtimes, entry.BuildDependencies)
		assert.Subset(t, expected.Buildtimes, entry.BuildDependencies,
			"Expected dependencies. Expected: %v, Actual, %v", expected.Buildtimes, entry.BuildDependencies)
	}
}
