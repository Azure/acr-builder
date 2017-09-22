package grok

import (
	"os"
	"path"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	mocks "github.com/Azure/acr-builder/tests/utils/domain"
	testutils "github.com/Azure/acr-builder/tests/utils/grok"
)

func TestParse(t *testing.T) {
	testResourceDir := path.Join("..", "..", "tests", "resources", "docker-compose")
	composeFile := path.Join(testResourceDir, "docker-compose-envs.yml")
	err := os.Setenv("ACR_BUILD_DOCKER_REGISTRY", "unit-tests")
	if err != nil {
		panic(err)
	}
	runner := new(mocks.MockRunner)
	runner.PrepareEnvResolves(map[string]string{
		"DOCKERFILE":                "Dockerfile",
		"ACR_BUILD_DOCKER_REGISTRY": "unit-tests",
	})
	actualDependencies, err := ResolveDockerComposeDependencies(runner, "", composeFile)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	testutils.AssertSameDependencies(t, []domain.ImageDependencies{
		testutils.HelloNodeExampleDependencies,
		testutils.MultistageExampleDependencies,
	}, actualDependencies)
}
