// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "strings"

// TrimQuote returns a slice of the string s, with all leading
// and trailing double or single quotes removed, as defined by Unicode.
func TrimQuotes(s string) string {
	return strings.Trim(s, "'\"")
}
