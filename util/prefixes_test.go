package util

import (
	"testing"
)

func TestPrefixRegistryToImageName(t *testing.T) {
	tests := []struct {
		registry string
		img      string
		expected string
	}{
		{"", "someimg", "someimg"},
		{"", "foo:latest", "foo:latest"},
		{"foo", "someimg", "foo/someimg"},
		{"foo", "someimg:bar", "foo/someimg:bar"},
		{"foo", "library/someimg:bar", "library/someimg:bar"},
	}

	for _, test := range tests {
		actual := PrefixRegistryToImageName(test.registry, test.img)
		if actual != test.expected {
			t.Errorf("expected %s, got %s", test.expected, actual)
		}
	}
}

func TestPrefixTags(t *testing.T) {
	tests := []struct {
		registry     string
		cmd          string
		expected     string
		expectedTags []string
	}{
		{"foo.azurecr.io", "build -f Dockerfile . -t test:latest --tag bar", "build -f Dockerfile . -t foo.azurecr.io/test:latest --tag foo.azurecr.io/bar", []string{"foo.azurecr.io/test:latest", "foo.azurecr.io/bar"}},
		{"", "build -t bar/foo:latest . --tag bar", "build -t bar/foo:latest . --tag bar", []string{"bar/foo:latest", "bar"}},
		{"foo.azurecr.io", "build -f Dockerfile . -t foo.azurecr.io/test:latest", "build -f Dockerfile . -t foo.azurecr.io/test:latest", []string{"foo.azurecr.io/test:latest"}},
		{"foo.azurecr.io", "build -f Dockerfile -t library/test:latest", "build -f Dockerfile -t library/test:latest", []string{"library/test:latest"}},
	}
	for _, test := range tests {
		actual, actualTags := PrefixTags(test.cmd, test.registry)
		if actual != test.expected {
			t.Errorf("expected %s, got %s", test.expected, actual)
		}
		if !StringSequenceEquals(actualTags, test.expectedTags) {
			t.Errorf("expected %v as tags, got %v", test.expectedTags, actualTags)
		}
	}
}
