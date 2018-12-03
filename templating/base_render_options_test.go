// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import "testing"

func TestParseValues_Valid(t *testing.T) {
	tests := []struct {
		values   []string
		expected string
	}{
		{
			[]string{"a=b", "b===ll", "c=12345", "d=ab=", "e=", "f=sadf=234"},
			`a: b
b: ==ll
c: "12345"
d: ab=
e: ""
f: sadf=234
`,
		},
		{
			[]string{"a=b", "a=c", "a=d"},
			`a: d
`,
		},
	}

	for _, test := range tests {
		actual, err := parseValues(test.values)
		if err != nil {
			t.Errorf("Failed to parse vals, err: %v", err)
		}
		if actual != test.expected {
			t.Errorf("Failed to parse values, expected '%s' but got '%s'", test.expected, actual)
		}
	}
}

func TestParseValues_Invalid(t *testing.T) {
	tests := []struct {
		values []string
	}{
		{[]string{"apple", "=k", "=====", "=", "", "           "}},
	}

	for _, test := range tests {
		if _, err := parseValues(test.values); err == nil {
			t.Errorf("Expected an error during parse values, but it was nil")
		}
	}
}

func TestParseRegistryName(t *testing.T) {
	tests := []struct {
		fullyQualifiedRegistryName string
		expectedRegistryName       string
	}{
		{"", ""},
		{"foo", "foo"},
		{"foo.azurecr.io", "foo"},
		{"foo-bar.azurecr-test.io", "foo-bar"},
		{"  ", "  "},
	}

	for _, test := range tests {
		if actual := parseRegistryName(test.fullyQualifiedRegistryName); actual != test.expectedRegistryName {
			t.Errorf("Expected %s but got %s for the registry name", test.expectedRegistryName, actual)
		}
	}
}
