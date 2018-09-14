// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"net/url"
	"strings"

	"github.com/docker/docker/pkg/urlutil"
)

const (
	azureDevOpsHost = "dev.azure.com"
	vstsHost        = ".visualstudio.com"
)

// IsAzureDevOpsGitURL determines whether or not the specified string is an Azure DevOps Git URL.
func IsAzureDevOpsGitURL(s string) bool {
	url, err := url.Parse(strings.ToLower(s))
	if err != nil {
		return false
	}
	return url.Scheme == "https" &&
		url.Host == azureDevOpsHost &&
		strings.Contains(url.Path, "/_git/") &&
		len(url.Query()) == 0
}

// IsVstsGitURL determines whether or not the specified string is a VSTS Git URL.
func IsVstsGitURL(s string) bool {
	url, err := url.Parse(strings.ToLower(s))
	if err != nil {
		return false
	}

	return url.Scheme == "https" &&
		strings.HasSuffix(url.Host, vstsHost) &&
		strings.Contains(url.Path, "/_git/") &&
		len(url.Query()) == 0
}

// IsSourceControlURL determines whether or not the specified string is a source control URL.
func IsSourceControlURL(s string) bool {
	return IsGitURL(s) || IsAzureDevOpsGitURL(s) || IsVstsGitURL(s)
}

// IsGitURL determines whether or not the specified string is a Git URL.
func IsGitURL(s string) bool {
	return urlutil.IsGitURL(s)
}

// IsURL determines whether or not the specified string is a URL.
func IsURL(s string) bool {
	return urlutil.IsURL(s)
}

// IsLocalContext determines whether or not the specified string is local.
func IsLocalContext(s string) bool {
	if IsURL(s) || IsSourceControlURL(s) {
		return false
	}
	return true
}
