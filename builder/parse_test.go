// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"testing"

	"github.com/Azure/acr-builder/util"
)

func TestParseRunArgs(t *testing.T) {
	tests := []struct {
		id       int
		cmd      string
		lookup   map[string]bool
		expected []string
	}{
		// Tag tests
		{1, "build -f Dockerfile -t {{.Build.ID}}:latest --tag blah https://github.com/Azure/acr-builder.git", tagLookup, []string{"{{.Build.ID}}:latest", "blah"}},
		{2, "build --tag foo https://github.com/Azure/acr-builder.git --tag bar -t qux", tagLookup, []string{"foo", "bar", "qux"}},

		// Build arg tests
		{3, "build -f Dockerfile -t hello:world --build-arg foo https://github.com/Azure/acr-builder.git --build-arg bar", buildArgLookup, []string{"foo", "bar"}},
		{4, "build -f Dockerfile -t hello:world --buildarg ignored --build-arg foo=bar https://github.com/Azure/acr-builder.git --build-arg hello=world", buildArgLookup, []string{"foo=bar", "hello=world"}},
	}

	for _, test := range tests {
		actual := parseRunArgs(test.cmd, test.lookup)
		if !util.StringSequenceEquals(actual, test.expected) {
			t.Errorf("Test %d failed. Expected %v, got %v", test.id, test.expected, actual)
		}
	}
}

// TestParseDockerBuildCmd tests stripping out the positional Docker context and Dockerfile name from a build command.
func TestParseDockerBuildCmd(t *testing.T) {
	tests := []struct {
		id         int
		cmd        string
		dockerfile string
		context    string
	}{
		{1, "build -f Dockerfile -t {{.Build.ID}}:latest https://github.com/Azure/acr-builder.git", "Dockerfile", "https://github.com/Azure/acr-builder.git"},
		{2, "build https://github.com/Azure/acr-builder.git -f Dockerfile -t foo:bar", "Dockerfile", "https://github.com/Azure/acr-builder.git"},
		{3, "build https://github.com/Azure/acr-builder.git#master:blah -f Dockerfile -t foo:bar", "Dockerfile", "https://github.com/Azure/acr-builder.git#master:blah"},
		{4, "build .", "Dockerfile", "."},
		{5, "build --file src/Dockerfile . -t foo:bar", "src/Dockerfile", "."},
		{6, "build -f src/Dockerfile .", "src/Dockerfile", "."},
		{7, "build -t foo https://github.com/Azure/acr-builder.git#:HelloWorld", "Dockerfile", "https://github.com/Azure/acr-builder.git#:HelloWorld"},
		// TODO: support reading from stdin?
		// {7, "build - < Dockerfile", "Dockerfile", "-"},
	}

	for _, test := range tests {
		dockerfile, context := parseDockerBuildCmd(test.cmd)
		if test.dockerfile != dockerfile {
			t.Errorf("Test %d failed. Expected %s as the dockerfile, got %s", test.id, test.dockerfile, dockerfile)
		}
		if test.context != context {
			t.Errorf("Test %d failed. Expected %s as the context, got %s", test.id, test.context, context)
		}
	}
}

// TestReplacePositionalContext tests replacing the positional context parameter in a build command.
func TestReplacePositionalContext(t *testing.T) {
	tests := []struct {
		cmd         string
		replacement string
		expected    string
	}{
		{"build -f Dockerfile -t blah:latest https://github.com/Azure/acr-builder.git", ".", "build -f Dockerfile -t blah:latest ."},
		{"build https://github.com/Azure/acr-builder.git -f Dockerfile -t foo:bar", "HelloWorld", "build HelloWorld -f Dockerfile -t foo:bar"},
		{"build .", "HelloWorld", "build HelloWorld"},
		{"build --file src/Dockerfile . -t foo:bar", "src/Dockerfile", "build --file src/Dockerfile src/Dockerfile -t foo:bar"},
		{"build -f src/Dockerfile .", "test/another-repo", "build -f src/Dockerfile test/another-repo"},
	}

	for _, test := range tests {
		if actual := replacePositionalContext(test.cmd, test.replacement); actual != test.expected {
			t.Errorf("Failed to replace positional context. Got %s, expected %s", actual, test.expected)
		}
	}
}

// TestGetContextFromGitURL tests getting context from a git URL.
func TestGetContextFromGitURL(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{"https://github.com/Azure/acr-builder.git#stable:.", "."},
		{"https://github.com/Azure/acr-builder.git#master:HelloWorld", "HelloWorld"},
		{"https://github.com/Azure/acr-builder.git#:Foo", "Foo"},
		{"https://github.com/Azure/acr-builder.git", "."},
		{"https://github.com/Azure/acr-builder.git#master", "."},
	}

	for _, test := range tests {
		if actual := getContextFromGitURL(test.cmd); actual != test.expected {
			t.Errorf("Failed to get context from git url. Got %s, expected %s", actual, test.expected)
		}
	}
}
