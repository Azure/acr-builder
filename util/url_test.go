// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "testing"

var (
	validGitURLs = []string{
		"https://github.com/Azure/acr-builder.git",
		"https://github.com/Azure/acr-builder.git#master:src",
		"https://github.com/Azure/acr-builder.git#:src",
		"https://github.com/Azure/acr-builder.git#master",
	}
	invalidGitURLs = []string{
		"https://github.com/Azure/acr-builder",
		"https://github.com/Azure/acr-builder.git#",
		"https://sometarcontext.com",
		"https://短.com",
	}
	validVSTSURLs = []string{
		"https://azure.visualstudio.com/ACR/_git/Build/",
		"https://PAT@azure.visualstudio.com/ACR/_git/Build/",
		"https://user:pat@azure.visualstudio.com/ACR/_git/Build/",
		"https://azure.visualstudio.com/ACR/_git/Build",
	}
	invalidVSTSURLs = []string{
		"   https://azure.visualstudio.com/ACR/_git/Build/", // leading spaces cause a parse error
		"https://短.com",
	}
	validDevOpsURLs = []string{
		"https://foo@dev.azure.com/foo/dockerfiles/_git/dockerfiles",
		"https://foo:bar@dev.azure.com/foo/dockerfiles/_git/dockerfiles",
		"https://dev.azure.com/foo/dockerfiles/_git/dockerfiles",
	}
	invalidDevOpsURLs = []string{
		"  https://foo@dev.azure.com/foo/dockerfiles/_git/dockerfiles", // leading spaces cause a parse error
		"https://短.com",
	}
)

func TestIsAzureDevOpsGitURL(t *testing.T) {
	invalidTests := validGitURLs
	invalidTests = append(invalidTests, invalidGitURLs...)
	invalidTests = append(invalidTests, validVSTSURLs...)
	invalidTests = append(invalidTests, invalidVSTSURLs...)
	invalidTests = append(invalidTests, invalidDevOpsURLs...)

	for _, test := range invalidTests {
		if valid := IsAzureDevOpsGitURL(test); valid {
			t.Errorf("%s should not be a devops url", test)
		}
	}

	for _, test := range validDevOpsURLs {
		if valid := IsAzureDevOpsGitURL(test); !valid {
			t.Errorf("%s should be a devops url", test)
		}
	}
}

func TestIsVstsGitURL(t *testing.T) {
	invalidTests := validGitURLs
	invalidTests = append(invalidTests, invalidGitURLs...)
	invalidTests = append(invalidTests, validDevOpsURLs...)
	invalidTests = append(invalidTests, invalidDevOpsURLs...)
	invalidTests = append(invalidTests, invalidVSTSURLs...)

	for _, test := range invalidTests {
		if valid := IsVstsGitURL(test); valid {
			t.Errorf("%s should not be a vsts url", test)
		}
	}

	for _, test := range validVSTSURLs {
		if valid := IsVstsGitURL(test); !valid {
			t.Errorf("%s should be a vsts url", test)
		}
	}
}

func TestIsGitURL(t *testing.T) {
	invalidTests := validVSTSURLs
	invalidTests = append(invalidTests, invalidVSTSURLs...)
	invalidTests = append(invalidTests, validDevOpsURLs...)
	invalidTests = append(invalidTests, invalidDevOpsURLs...)
	invalidTests = append(invalidTests, invalidGitURLs...)

	for _, test := range invalidTests {
		if valid := IsGitURL(test); valid {
			t.Errorf("%s should not be a vsts url", test)
		}
	}

	for _, test := range validGitURLs {
		if valid := IsGitURL(test); !valid {
			t.Errorf("%s should be a vsts url", test)
		}
	}
}

func TestIsSourceControlURL(t *testing.T) {
	validTests := validDevOpsURLs
	validTests = append(validTests, validVSTSURLs...)
	validTests = append(validTests, validGitURLs...)

	for _, test := range validTests {
		if valid := IsSourceControlURL(test); !valid {
			t.Errorf("%s should be a valid source control url", test)
		}
	}

	invalidTests := invalidDevOpsURLs
	invalidTests = append(invalidTests, invalidVSTSURLs...)
	invalidTests = append(invalidTests, invalidGitURLs...)

	for _, test := range invalidTests {
		if valid := IsSourceControlURL(test); valid {
			t.Errorf("%s shouldn't be a valid source control url", test)
		}
	}
}

func TestIsLocalContext(t *testing.T) {
	invalidTests := validDevOpsURLs
	invalidTests = append(invalidTests, validGitURLs...)
	invalidTests = append(invalidTests, validVSTSURLs...)

	for _, test := range invalidTests {
		if valid := IsLocalContext(test); valid {
			t.Errorf("%s should not be a valid local context", test)
		}
	}

	tests := []struct {
		url      string
		expected bool
	}{
		{"baseimages/foo", true},
		{".", true},
	}

	for _, test := range tests {
		if actual := IsLocalContext(test.url); actual != test.expected {
			t.Errorf("Expected %v for local ctx: %s, but got %v", test.expected, test.url, actual)
		}
	}
}
