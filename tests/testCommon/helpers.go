package testCommon

import (
	"strconv"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/stretchr/testify/assert"
)

// NilError is a placeholder for mock objects to use when mocking a function that returns an error
var NilError error

// EmptyContext is an empty build context
var EmptyContext = build.NewContext([]build.EnvVar{}, []build.EnvVar{})

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
var MultiStageExampleTestEnv = []build.EnvVar{
	{Name: constants.ExportsDockerRegistry, Value: TestsDockerRegistryName},
	// this is just a goofy entry to show that variable resolutions work
	{Name: "DOCKERFILE", Value: "Dockerfile"},
}

// MultistageExampleDependencies returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-multistage with default docker registry name
var MultistageExampleDependencies = MultistageExampleDependenciesOn(TestsDockerRegistryName)

// HelloNodeExampleDependencies returns dependencies to the project in  ${workspaceRoot}/tests/resources/docker-compose/hello-node
var HelloNodeExampleDependencies = HelloNodeExampleDependenciesOn(TestsDockerRegistryName)

// MultistageExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-multistage
func MultistageExampleDependenciesOn(registry string) build.ImageDependencies {
	return *NewImageDependencies(
		registry+"hello-multistage",
		"alpine",
		[]string{"golang:alpine"},
	)
}

// HelloNodeExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/docker-compose/hello-node
func HelloNodeExampleDependenciesOn(registry string) build.ImageDependencies {
	return *NewImageDependencies(
		registry+"hello-node",
		"node:alpine",
		[]string{},
	)
}

// AssertSameDependencies help determine two sets of dependencies are equivalent
func AssertSameDependencies(t *testing.T, expectedList []build.ImageDependencies, actualList []build.ImageDependencies) {
	assert.Equal(t, len(expectedList), len(actualList), "Unexpected numbers of image dependencies")
	expectedMap := map[string]build.ImageDependencies{}
	for _, entry := range expectedList {
		expectedMap[entry.Image.String()] = entry
	}
	for _, entry := range actualList {
		expected, found := expectedMap[entry.Image.String()]
		assert.True(t, found, "Unexpected image name: %s", entry.Image)
		assert.Equal(t, expected.Runtime, entry.Runtime)
		assert.Equal(t, len(expected.Buildtime), len(entry.Buildtime),
			"Incorrect number of dependencies. Expected: %v, Actual, %v", expected.Buildtime, entry.Buildtime)

		expectSubMap := map[string]build.ImageReference{}
		for _, buildtime := range expected.Buildtime {
			expectSubMap[buildtime.String()] = *buildtime
		}
		for _, actualReference := range entry.Buildtime {
			expectedBuildTime := expectSubMap[actualReference.String()]
			assert.Equal(t, expectedBuildTime, *actualReference)
		}
	}
}

// DotnetExampleTargetRegistryName is a placeholder registry name
const DotnetExampleTargetRegistryName = "registry"

// DotnetExampleTargetImageName is a placeholder image name
const DotnetExampleTargetImageName = "img"

// DotnetExampleFullImageName is the image name for dotnet example
const DotnetExampleFullImageName = DotnetExampleTargetRegistryName + "/" + DotnetExampleTargetImageName

// DotnetExampleDependencies links to the project in ${workspaceRoot}/tests/resources/docker-dotnet
var DotnetExampleDependencies = *NewImageDependencies(
	DotnetExampleFullImageName,
	"microsoft/aspnetcore:2.0",
	[]string{"microsoft/aspnetcore-build:2.0", "imaginary/cert-generator:1.0"},
)

// AssertSameEnv asserts two sets environment variable are the same
func AssertSameEnv(t *testing.T, expected, actual []build.EnvVar) {
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

// ReportOnError Reports an error when function fails
func ReportOnError(t *testing.T, f func() error) {
	err := f()
	if err != nil {
		t.Error(err.Error())
	}
}

// NewImageDependencies creates a image dependency object
func NewImageDependencies(image string, runtime string, buildtimes []string) *build.ImageDependencies {
	dep, err := build.NewImageDependencies(EmptyContext, image, runtime, buildtimes)
	if err != nil {
		panic(err)
	}
	return dep
}

// GetDigest gets the mock digest of a image
func GetDigest(image string) string {
	return "sha-" + image
}

// DependenciesWithDigests populates mock digest values for image dependencies
func DependenciesWithDigests(original build.ImageDependencies) *build.ImageDependencies {
	result := &build.ImageDependencies{
		Image:   ImageReferenceWithDigest(original.Image),
		Runtime: ImageReferenceWithDigest(original.Runtime),
	}
	for _, buildtime := range original.Buildtime {
		result.Buildtime = append(result.Buildtime, ImageReferenceWithDigest(buildtime))
	}
	return result
}

// ImageReferenceWithDigest populates mock digest values for image
func ImageReferenceWithDigest(original *build.ImageReference) *build.ImageReference {
	imageName := original.String()
	result, err := build.NewImageReference(imageName)
	if err != nil {
		panic("Failed to process " + imageName)
	}
	result.Digest = GetDigest(imageName)
	return result
}
