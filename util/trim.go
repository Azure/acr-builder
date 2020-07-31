// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "strings"

// TrimQuotes returns a slice of the string s, with all leading
// and trailing double or single quotes removed, as defined by Unicode.
func TrimQuotes(s string) string {
	return strings.Trim(s, "'\"")
}

// TrimArtifactPrefix returns a slice of the string s, with leading
// artifact prefix removed
func TrimArtifactPrefix(s string) string {
	if strings.HasPrefix(strings.ToLower(s), "oci://") && len(s) > 6 {
		return s[6:]
	}
	return s
}
