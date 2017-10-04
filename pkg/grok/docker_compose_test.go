package grok

import (
	"path/filepath"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"

	"github.com/Azure/acr-builder/pkg/domain"
	testutils "github.com/Azure/acr-builder/tests/testCommon"
)

func TestParse(t *testing.T) {
	testResourceDir := filepath.Join("..", "..", "tests", "resources", "docker-compose")
	composeFile := filepath.Join(testResourceDir, "docker-compose-envs.yml")
	env := domain.NewContext([]domain.EnvVar{
		{Name: constants.ExportsDockerRegistry, Value: "unit-tests"},
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
