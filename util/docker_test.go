package util

import "testing"

// TestParseDockerBuildCmd tests stripping out the positional Docker context and Dockerfile name from a build command.
func TestParseDockerBuildCmd(t *testing.T) {
	type parseTest struct {
		id         int
		cmd        string
		dockerfile string
		context    string
	}

	tests := []parseTest{
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
		dockerfile, context := ParseDockerBuildCmd(test.cmd)
		if test.dockerfile != dockerfile {
			t.Errorf("Test %d failed. Expected %s as the dockerfile, got %s", test.id, test.dockerfile, dockerfile)
		}
		if test.context != context {
			t.Errorf("Test %d failed. Expected %s as the context, got %s", test.id, test.context, context)
		}
	}
}
