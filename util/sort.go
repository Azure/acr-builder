// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "strings"

// SortablePathLen allows for sorting an array of strings by their path length.
type SortablePathLen []string

// Len returns the length of the SortablePathLen.
func (s SortablePathLen) Len() int {
	return len(s)
}

// Swap swaps two values in the SortablePathLen.
func (s SortablePathLen) Swap(i, j int) {
	s[j], s[i] = s[i], s[j]
}

// Less compares two values in the SortablePathLen.
func (s SortablePathLen) Less(i, j int) bool {
	a, b := s[i], s[j]
	ca, cb := strings.Count(a, "/"), strings.Count(b, "/")
	if ca == cb {
		return strings.Compare(a, b) == -1
	}
	return ca < cb
}
