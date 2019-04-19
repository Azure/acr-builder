// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"testing"
)

// TestParseDockerBuildCmd tests stripping out the positional Docker context and Dockerfile name from a build command.
func TestParseDockerBuildCmd(t *testing.T) {
	tests := []struct {
		id         int
		build      string
		dockerfile string
		context    string
	}{
		{1, "-f Dockerfile -t {{.Run.ID}}:latest https://github.com/Azure/acr-builder.git", "Dockerfile", "https://github.com/Azure/acr-builder.git"},
		{2, "https://github.com/Azure/acr-builder.git -f Dockerfile -t foo:bar", "Dockerfile", "https://github.com/Azure/acr-builder.git"},
		{3, "https://github.com/Azure/acr-builder.git#master:blah -f Dockerfile -t foo:bar", "Dockerfile", "https://github.com/Azure/acr-builder.git#master:blah"},
		{4, ".", "", "."},
		{5, "--file src/Dockerfile . -t foo:bar", "src/Dockerfile", "."},
		{6, "-f src/Dockerfile .", "src/Dockerfile", "."},
		{7, "-t foo https://github.com/Azure/acr-builder.git#:HelloWorld", "", "https://github.com/Azure/acr-builder.git#:HelloWorld"},
	}

	for _, test := range tests {
		dockerfile, context := parseDockerBuildCmd(test.build)
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
		build       string
		replacement string
		expected    string
	}{
		{"-f Dockerfile -t blah:latest https://github.com/Azure/acr-builder.git", ".", "-f Dockerfile -t blah:latest ."},
		{"https://github.com/Azure/acr-builder.git -f Dockerfile -t foo:bar", "HelloWorld", "HelloWorld -f Dockerfile -t foo:bar"},
		{".", "HelloWorld", "HelloWorld"},
		{"--file src/Dockerfile . -t foo:bar", "src/Dockerfile", "--file src/Dockerfile src/Dockerfile -t foo:bar"},
		{"-f src/Dockerfile .", "test/another-repo", "-f src/Dockerfile test/another-repo"},
		{"-f src/Dockerfile https://foo.visualstudio.com/ACR/_git/Build/#master:.", "vstsreplacement", "-f src/Dockerfile vstsreplacement"},
		{"", "apple", ""}, // Nothing to replace
	}

	for _, test := range tests {
		if actual := replacePositionalContext(test.build, test.replacement); actual != test.expected {
			t.Errorf("Failed to replace positional context. Got %s, expected %s", actual, test.expected)
		}
	}
}

// TestGetContextFromGitURL tests getting context from a git URL.
func TestGetContextFromGitURL(t *testing.T) {
	tests := []struct {
		build    string
		expected string
	}{
		// GitHub
		{"https://github.com/Azure/acr-builder.git#stable:.", "."},
		{"https://github.com/Azure/acr-builder.git#master:HelloWorld", "HelloWorld"},
		{"https://github.com/Azure/acr-builder.git#:Foo", "Foo"},
		{"https://github.com/Azure/acr-builder.git", "."},
		{"https://github.com/Azure/acr-builder.git#master", "."},

		// VSO
		{"https://foo.visualstudio.com/ACR/_git/Build/#master:.", "."},
		{"https://foo.visualstudio.com/ACR/_git/Build/#master:HelloWorld", "HelloWorld"},
		{"https://foo.visualstudio.com/ACR/_git/Build/#:Foo", "Foo"},
		{"https://foo.visualstudio.com/ACR/_git/Build/", "."},
		{"https://foo.visualstudio.com/ACR/_git/Build/#master", "."},
		{"https://foo.VISUALstuDIo.com/ACR/_git/Build#master:HelloWorld", "HelloWorld"},

		// Azure Devops
		{"https://dev.azure.com/foo/_git/dockerfiles#stable:.", "."},
		{"https://dev.azure.com/foo/_git/dockerfiles#master:HelloWorld", "HelloWorld"},
		{"https://dev.azure.com/foo/_git/dockerfiles#:Foo", "Foo"},
		{"https://dev.azure.com/foo/_git/dockerfiles", "."},
		{"https://dev.azure.com/foo/_git/dockerfiles#branch", "."},
	}

	for _, test := range tests {
		if actual := getContextFromGitURL(test.build); actual != test.expected {
			t.Errorf("Failed to get context from git url. Got %s, expected %s", actual, test.expected)
		}
	}
}
