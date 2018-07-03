// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

// IsMap determines whether the provided interface is a map.
func IsMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}
