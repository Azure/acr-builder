// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import (
	"strings"
)

// Errors is simply a wrapper for an array of errors. Useful for aggregating
// exceptions while in a loop if you don't want to early return.
type Errors []error

// String returns a string representation for the Errors type.
func (e Errors) String() string {
	if len(e) == 0 {
		return ""
	}

	out := make([]string, len(e))
	for i := range e {
		out[i] = e[i].Error()
	}

	return strings.Join(out, ", ")
}
