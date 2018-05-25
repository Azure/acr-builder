package util

import (
	"net/url"
	"strings"
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
