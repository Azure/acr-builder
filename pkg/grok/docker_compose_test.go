package grok

import (
	"path"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
)

func TestParse(t *testing.T) {
	testResourceDir := path.Join("..", "..", "tests", "resources", "docker-compose")
	composeFile := path.Join(testResourceDir, "docker-compose-envs.yml")
	env := domain.NewContext([]domain.EnvVar{
		{Name: "ACR_BUILD_DOCKER_REGISTRY", Value: "unit-tests"},
	}, []domain.EnvVar{
		{Name: "DOCKERFILE", Value: "Dockerfile"},
	})
	actualDependencies, err := ResolveDockerComposeDependencies(env, "", composeFile)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	testutils.AssertSameDependencies(t, []domain.ImageDependencies{
		testutils.HelloNodeExampleDependencies,
		testutils.MultistageExampleDependencies,
	}, actualDependencies)
}
