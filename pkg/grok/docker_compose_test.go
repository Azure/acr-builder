package grok

import (
	"path"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	mocks "github.com/Azure/acr-builder/tests/utils/domain"
	testutils "github.com/Azure/acr-builder/tests/utils/grok"
)

func TestParse(t *testing.T) {
	testResourceDir := path.Join("..", "..", "tests", "resources", "docker-compose")
	composeFile := path.Join(testResourceDir, "docker-compose-envs.yml")
	runner := new(mocks.MockRunner)
	runner.PrepareResolve([]*mocks.ArgumentResolution{
		mocks.ResolveDefault("./hello-multistage"),
		mocks.Resolve("${ACR_BUILD_DOCKER_REGISTRY}/hello-multistage", "unit-tests/hello-multistage"),
		mocks.ResolveDefault("./hello-node"),
		mocks.Resolve("$DOCKERFILE.alpine", "Dockerfile.alpine"),
		mocks.Resolve("${ACR_BUILD_DOCKER_REGISTRY}/hello-node", "unit-tests/hello-node"),
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
