// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"net/url"
	"strings"

	"github.com/docker/docker/pkg/urlutil"
)

// IsVstsGitURL determines whether or not the specified string is a VSTS Git URL.
func IsVstsGitURL(s string) bool {
	url, err := url.Parse(strings.ToLower(s))
	if err != nil {
		return false
	}

	return url.Scheme == "https" &&
		strings.HasSuffix(url.Host, ".visualstudio.com") &&
		strings.Contains(url.Path, "/_git/") &&
		len(url.Query()) == 0
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
	if IsVstsGitURL(s) || IsGitURL(s) || IsURL(s) {
		return false
	}

	return true
}
