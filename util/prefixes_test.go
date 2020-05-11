// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"testing"
)

func TestPrefixRegistryToImageName(t *testing.T) {
	tests := []struct {
		registry           string
		img                string
		allKnownRegistries []string
		expected           string
	}{
		{"", "someimg", []string{""}, "someimg"},
		{"", "foo:latest", []string{""}, "foo:latest"},
		{"foo", "someimg", []string{"foo"}, "foo/someimg"},
		{"foo", "someimg:bar", []string{"foo"}, "foo/someimg:bar"},
		{"foo", "library/someimg:bar", []string{"foo"}, "library/someimg:bar"},
		{"foo", "library/someimg:bar", []string{"foo"}, "library/someimg:bar"},
		{"foo", "bar/someimage:latest", []string{"foo", "bar"}, "bar/someimage:latest"},
	}

	for _, test := range tests {
		actual := PrefixRegistryToImageName(test.registry, test.img, test.allKnownRegistries)
		if actual != test.expected {
			t.Errorf("expected %s, got %s", test.expected, actual)
		}
	}
}

func TestPrefixTags(t *testing.T) {
	tests := []struct {
		registry           string
		allKnownRegistries []string
		cmd                string
		expected           string
		expectedTags       []string
	}{
		{
			"foo.azurecr.io",
			[]string{"foo.azurecr.io"},
			"build -f Dockerfile . -t test:latest --tag bar",
			"build -f Dockerfile . -t foo.azurecr.io/test:latest --tag foo.azurecr.io/bar",
			[]string{"foo.azurecr.io/test:latest", "foo.azurecr.io/bar"},
		},
		{
			"",
			[]string{""},
			"build -t bar/foo:latest . --tag bar",
			"build -t bar/foo:latest . --tag bar",
			[]string{"bar/foo:latest", "bar"},
		},
		{
			"foo.azurecr.io",
			[]string{"foo.azurecr.io"},
			"build -f Dockerfile . -t foo.azurecr.io/test:latest",
			"build -f Dockerfile . -t foo.azurecr.io/test:latest",
			[]string{"foo.azurecr.io/test:latest"},
		},
		{
			"sample.azurecr.io",
			[]string{"sample.azurecr.io"},
			"build -f src/Dockerfile https://github.com/Azure/acr-builder.git -t testing/sub/repo:l",
			"build -f src/Dockerfile https://github.com/Azure/acr-builder.git -t sample.azurecr.io/testing/sub/repo:l",
			[]string{"sample.azurecr.io/testing/sub/repo:l"},
		},
		{
			"foo.azurecr.io",
			[]string{"foo.azurecr.io"},
			"build -f Dockerfile -t library/test:latest",
			"build -f Dockerfile -t library/test:latest",
			[]string{"library/test:latest"},
		},
		{
			"foo.azurecr.io",
			[]string{"foo.azurecr.io", "bar.azurecr.io"},
			"build -f Dockerfile -t bar.azurecr.io/test:latest",
			"build -f Dockerfile -t bar.azurecr.io/test:latest",
			[]string{"bar.azurecr.io/test:latest"},
		},
	}
	for _, test := range tests {
		actual, actualTags := PrefixTags(test.cmd, test.registry, test.allKnownRegistries)
		if actual != test.expected {
			t.Errorf("expected %s, got %s", test.expected, actual)
		}
		if !StringSequenceEquals(actualTags, test.expectedTags) {
			t.Errorf("expected %v as tags, got %v", test.expectedTags, actualTags)
		}
	}
}
