// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

//go:build linux || darwin

package builder

func getShell() []string {
	return []string{"/bin/sh", "-c"}
}
