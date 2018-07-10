package util

import (
	"testing"
)

// TestParseTags tests parsing tags off a build command.
func TestParseTags(t *testing.T) {
	tests := []struct {
		id       int
		cmd      string
		expected []string
	}{
		// Tag tests
		{1, "build -f Dockerfile -t {{.Build.ID}}:latest --tag blah https://github.com/Azure/acr-builder.git", []string{"{{.Build.ID}}:latest", "blah"}},
		{2, "build --tag foo https://github.com/Azure/acr-builder.git --tag bar -t qux", []string{"foo", "bar", "qux"}},
	}

	for _, test := range tests {
		actual := ParseTags(test.cmd)
		if !StringSequenceEquals(actual, test.expected) {
			t.Errorf("Test %d failed. Expected %v, got %v", test.id, test.expected, actual)
		}
	}
}

// TestParseBuildArgs tests parsing build args off a build command.
func TestParseBuildArgs(t *testing.T) {
	tests := []struct {
		id       int
		cmd      string
		expected []string
	}{
		{1, "build -f Dockerfile -t hello:world --build-arg foo https://github.com/Azure/acr-builder.git --build-arg bar", []string{"foo", "bar"}},
		{2, "build -f Dockerfile -t hello:world --buildarg ignored --build-arg foo=bar https://github.com/Azure/acr-builder.git --build-arg hello=world", []string{"foo=bar", "hello=world"}},
	}

	for _, test := range tests {
		actual := ParseBuildArgs(test.cmd)
		if !StringSequenceEquals(actual, test.expected) {
			t.Errorf("Test %d failed. Expected %v, got %v", test.id, test.expected, actual)
		}
	}
}
