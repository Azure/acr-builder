// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "github.com/google/uuid"

// IsValidUUID returns true if the specified string is a valid uuid,
// false otherwise.
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
