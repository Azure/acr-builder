package testCommon

import (
	"strconv"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/stretchr/testify/assert"
)

// NilError is a placeholder for mock objects to use when mocking a function that returns an error
var NilError error

// StringGenerator generate unique string with given prefix
type StringGenerator struct {
	Prefix  string
	counter int
}

// Next get next unique value
func (g *StringGenerator) Next() string {
	g.counter++
	return g.Prefix + strconv.Itoa(g.counter)
}

// MappedStringGenerator geneate unique strings and allow lookup for string that was generated
type MappedStringGenerator struct {
	StringGenerator
	lookup map[string]string
}

// NewMappedGenerator generates a unique string and can be looked up later by the given key value
func NewMappedGenerator(prefix string) *MappedStringGenerator {
	return &MappedStringGenerator{
		StringGenerator: StringGenerator{Prefix: prefix},
		lookup:          make(map[string]string),
	}
}

// NextWithKey generates a unique string and can be looked up later by the given key value
func (g *MappedStringGenerator) NextWithKey(key string) string {
	value := g.Next()
	g.lookup[key] = value
	return value
}

// Lookup the unique string generated
func (g *MappedStringGenerator) Lookup(key string) string {
	return g.lookup[key]
}

// TestsDockerRegistryName is the registry name used for testing
const TestsDockerRegistryName = "unit-tests/"

// MultiStageExampleTestEnv is the test env used for resolving docker-compose{-envs}.yml
var MultiStageExampleTestEnv = []domain.EnvVar{
	{Name: constants.ExportsDockerRegistry, Value: TestsDockerRegistryName},
	// this is just a goofy entry to show that variable resolutions work
	{Name: "DOCKERFILE", Value: "Dockerfile"},
}

// MultistageExampleDependencies returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-multistage with default docker registry name
var MultistageExampleDependencies = MultistageExampleDependenciesOn(TestsDockerRegistryName)

// HelloNodeExampleDependencies returns dependencies to the project in  ${workspaceRoot}/tests/resources/docker-compose/hello-node
var HelloNodeExampleDependencies = HelloNodeExampleDependenciesOn(TestsDockerRegistryName)

// MultistageExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-multistage
func MultistageExampleDependenciesOn(registry string) domain.ImageDependencies {
	return domain.ImageDependencies{
		Image:             registry + "hello-multistage",
		RuntimeDependency: "alpine",
		BuildDependencies: []string{"golang:alpine"},
	}
}

// HelloNodeExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-node
func HelloNodeExampleDependenciesOn(registry string) domain.ImageDependencies {
	return domain.ImageDependencies{
		Image:             registry + "hello-node",
		RuntimeDependency: "node:alpine",
		BuildDependencies: []string{},
	}
}

// AssertSameDependencies help determine two sets of dependencies are equivalent
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

// DotnetExampleTargetRegistryName is a placeholder registry name
const DotnetExampleTargetRegistryName = "registry"

// DotnetExampleTargetImageName is a placeholder image name
const DotnetExampleTargetImageName = "img"

// DotnetExampleDependencies links to the project in ${workspaceRoot}/tests/resources/docker-dotnet
var DotnetExampleDependencies = domain.ImageDependencies{
	Image:             DotnetExampleTargetRegistryName + "/" + DotnetExampleTargetImageName,
	RuntimeDependency: "microsoft/aspnetcore:2.0",
	BuildDependencies: []string{"microsoft/aspnetcore-build:2.0", "imaginary/cert-generator:1.0"},
}

// AssertSameEnv asserts two sets environment variable are the same
func AssertSameEnv(t *testing.T, expected, actual []domain.EnvVar) {
	assert.Equal(t, len(expected), len(actual))
	env := map[string]string{}
	for _, entry := range expected {
		env[entry.Name] = entry.Value
	}
	for _, entry := range actual {
		value, found := env[entry.Name]
		assert.True(t, found, "key %s not found", entry.Name)
		assert.Equal(t, value, entry.Value, "key %s, expected: %s, actual: %s", entry.Name, value, entry.Value)
	}
}
