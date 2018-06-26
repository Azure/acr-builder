package builder

import "testing"

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
		{"https://github.com/Azure/acr-builder.git", "."},
		{"https://github.com/Azure/acr-builder.git#master", "."},
	}

	for _, test := range tests {
		if actual := getContextFromGitURL(test.cmd); actual != test.expected {
			t.Errorf("Failed to get context from git url. Got %s, expected %s", actual, test.expected)
		}
	}
}
