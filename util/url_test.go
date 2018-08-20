// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

func TestIsVstsGitURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://azure.visualstudio.com/ACR/_git/Build/", true},
		{"https://azure.visualstudio.com/ACR/_git/Build", true},
		{"https://github.com/Azure/acr-builder", false},
		{"https://github.com/Azure/acr-builder.git", false},
	}

	for _, test := range tests {
		if actual := IsVstsGitURL(test.url); actual != test.expected {
			t.Errorf("Expected %v for vsts url: %s, but got %v", test.expected, test.url, actual)
		}
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://azure.visualstudio.com/ACR/_git/Build/", false},
		{"https://azure.visualstudio.com/ACR/_git/Build", false},
		{"https://github.com/Azure/acr-builder", false},
		{"https://github.com/Azure/acr-builder.git#", false},
		{"https://github.com/Azure/acr-builder.git", true},
		{"https://github.com/Azure/acr-builder.git#master:src", true},
		{"https://github.com/Azure/acr-builder.git#:src", true},
		{"https://github.com/Azure/acr-builder.git#master", true},
	}

	for _, test := range tests {
		if actual := IsGitURL(test.url); actual != test.expected {
			t.Errorf("Expected %v for git url: %s, but got %v", test.expected, test.url, actual)
		}
	}
}

func TestIsLocalContext(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"baseimages/foo", true},
		{".", true},
		{"https://azure.visualstudio.com/ACR/_git/Build/", false},
		{"https://azure.visualstudio.com/ACR/_git/Build", false},
		{"https://sometarcontext.com", false},
		{"https://github.com/Azure/acr-builder", false},
		{"https://github.com/Azure/acr-builder.git", false},
	}

	for _, test := range tests {
		if actual := IsLocalContext(test.url); actual != test.expected {
			t.Errorf("Expected %v for local ctx: %s, but got %v", test.expected, test.url, actual)
		}
	}
}
