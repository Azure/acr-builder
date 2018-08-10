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
