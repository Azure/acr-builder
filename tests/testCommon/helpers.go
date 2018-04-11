package testCommon

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
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

// MultistageExampleDependencies returns dependencies to the project in ${workspaceRoot}/tests/resources/hello-multistage with default docker registry name
var MultistageExampleDependencies = MultistageExampleDependenciesOn(TestsDockerRegistryName)

// HelloNodeExampleDependencies returns dependencies to the project in  ${workspaceRoot}/tests/resources/hello-node
var HelloNodeExampleDependencies = HelloNodeExampleDependenciesOn(TestsDockerRegistryName)

// MultistageExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/hello-multistage
func MultistageExampleDependenciesOn(registry string) build.ImageDependencies {
	return *NewImageDependencies(
		registry+"hello-multistage",
		"alpine",
		[]string{"golang:alpine"},
	)
}

// HelloNodeExampleDependenciesOn returns dependencies to the project in ${workspaceRoot}/tests/resources/hello-node
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
		if entry.Image != nil {
			expectedMap[entry.Image.String()] = entry
		} else {
			expectedMap[""] = entry
		}
	}
	for _, entry := range actualList {
		var expected build.ImageDependencies
		var found bool

		if entry.Image != nil {
			expected, found = expectedMap[entry.Image.String()]
		} else {
			expected, found = expectedMap[""]
		}

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

func AssertErrorPattern(t *testing.T, pattern string, err error) {
	if pattern != "" {
		assert.Regexp(t, regexp.MustCompile(pattern), err.Error())
	} else {
		assert.Nil(t, err)
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
	"microsoft/aspnetcore:2.0.6",
	[]string{"microsoft/aspnetcore-build:2.0", "imaginary/cert-generator:1.0"},
)

// DotnetExampleMinimalBuildArg is the minimal build arg required for dotnet example
var DotnetExampleMinimalBuildArg = []string{"build_image=aspnetcore-build"}

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

// GetRepoDigests gets the mock RepoDigests of a image
func GetRepoDigests(image string) string {
	repo := strings.Split(image, ":")[0]
	return "[\"" + repo + "@sha256:" + image + "\"]"
}

// GetDigest gets the mock digest of a image
func GetDigest(image string) string {
	return "sha256:" + image
}

// DependenciesWithDigests populates mock digest values for image dependencies
func DependenciesWithDigests(original build.ImageDependencies) *build.ImageDependencies {
	result := &build.ImageDependencies{
		Runtime: ImageReferenceWithDigest(original.Runtime),
	}
	if original.Image != nil {
		result.Image = ImageReferenceWithDigest(original.Image)
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

var ExpectedACRBuilderRelativePaths = []string{
	"Dockerfile",
	"CONTRIBUTING.md",
	"Makefile",
	"main.go",
	"README.md",
	"VERSION",
	"LICENSE",
}

var ExpectedMultiStageExampleRelativePaths = []string{
	"Dockerfile",
	"program.go",
}
