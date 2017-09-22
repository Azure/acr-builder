package grok

import (
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/shhsu/testify/assert"
)

var MultistageExampleDependencies = domain.ImageDependencies{
	Image:             "unit-tests/hello-multistage",
	RuntimeDependency: "alpine",
	BuildDependencies: []string{"golang:alpine"},
}

var HelloNodeExampleDependencies = domain.ImageDependencies{
	Image:             "unit-tests/hello-node",
	RuntimeDependency: "node:alpine",
	BuildDependencies: []string{},
}

func AssertSameDependencies(t *testing.T, expectedList []domain.ImageDependencies, actual []domain.ImageDependencies) {
	assert.Equal(t, len(expectedList), len(actual), "Unexpected numbers of image dependencies")
	expectedMap := map[string]domain.ImageDependencies{}
	for _, entry := range expectedList {
		expectedMap[entry.Image] = entry
	}

	for _, entry := range actual {
		expected, found := expectedMap[entry.Image]
		assert.True(t, found, "Unexpected image name: %s", entry.Image)
		assert.Equal(t, expected.RuntimeDependency, entry.RuntimeDependency)
		assert.Equal(t, len(expected.BuildDependencies), len(entry.BuildDependencies),
			"Incorrect number of runtime dependencies. Expected: %v, Actual, %v", expected.BuildDependencies, entry.BuildDependencies)
		assert.Subset(t, expected.BuildDependencies, entry.BuildDependencies,
			"Expected dependencies. Expected: %v, Actual, %v", expected.BuildDependencies, entry.BuildDependencies)
	}
}
