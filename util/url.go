// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"net/url"
	"regexp"
	"strings"
)

const (
	azureDevOpsHost = "dev.azure.com"
	vstsHost        = ".visualstudio.com"
	httpsScheme     = "https"
)

// gitURLPathWithFragmentSuffix matches fragments to use as Git reference and build
// context from the Git repository. See IsGitURL for details.
var gitURLPathWithFragmentSuffix = regexp.MustCompile(`\.git(?:#.+)?$`)

// IsAzureDevOpsGitURL determines whether or not the specified string is an Azure DevOps Git URL.
func IsAzureDevOpsGitURL(s string) bool {
	lowercaseURL, err := url.Parse(strings.ToLower(s))
	if err != nil {
		return false
	}
	return lowercaseURL.Scheme == httpsScheme &&
		lowercaseURL.Host == azureDevOpsHost &&
		strings.Contains(lowercaseURL.Path, "/_git/") &&
		len(lowercaseURL.Query()) == 0
}

// IsVstsGitURL determines whether or not the specified string is a VSTS Git URL.
func IsVstsGitURL(s string) bool {
	lowercaseURL, err := url.Parse(strings.ToLower(s))
	if err != nil {
		return false
	}

	return lowercaseURL.Scheme == httpsScheme &&
		strings.HasSuffix(lowercaseURL.Host, vstsHost) &&
		strings.Contains(lowercaseURL.Path, "/_git/") &&
		len(lowercaseURL.Query()) == 0
}

// IsSourceControlURL determines whether or not the specified string is a source control URL.
func IsSourceControlURL(s string) bool {
	return IsGitURL(s) || IsAzureDevOpsGitURL(s) || IsVstsGitURL(s)
}

// IsGitURL determines whether or not the specified string is a Git URL.
//
// This is a rudimentary check (no validation is performed to ensure the URL
// is well-formed) mirroring docker build's own git-context detection:
// https://github.com/moby/moby/blob/master/builder/remotecontext/urlutil/urlutil.go
func IsGitURL(s string) bool {
	if IsURL(s) && gitURLPathWithFragmentSuffix.MatchString(s) {
		return true
	}
	for _, prefix := range []string{"git://", "github.com/", "git@"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// IsRegistryArtifact determines whether or not the specified string is a registry artifact
func IsRegistryArtifact(s string) bool {
	return strings.HasPrefix(strings.ToLower(s), "oci://")
}

// IsURL determines whether or not the specified string is a URL.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

// IsLocalContext determines whether or not the specified string is local.
func IsLocalContext(s string) bool {
	if IsURL(s) || IsSourceControlURL(s) || IsRegistryArtifact(s) {
		return false
	}
	return true
}
