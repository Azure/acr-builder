package grok

import (
	"path/filepath"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/tests/testCommon"

	"github.com/Azure/acr-builder/pkg/domain"
)

func TestParse(t *testing.T) {
	testResourceDir := filepath.Join("..", "..", "tests", "resources", "docker-compose")
	composeFile := filepath.Join(testResourceDir, "docker-compose-envs.yml")
	env := domain.NewContext([]domain.EnvVar{
		{Name: constants.ExportsDockerRegistry, Value: testCommon.TestsDockerRegistryName},
	}, []domain.EnvVar{
		{Name: "DOCKERFILE", Value: "Dockerfile"},
	})
	actualDependencies, err := ResolveDockerComposeDependencies(env, "", composeFile)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	testCommon.AssertSameDependencies(t, []domain.ImageDependencies{
		testCommon.HelloNodeExampleDependencies,
		testCommon.MultistageExampleDependencies,
	}, actualDependencies)
}
