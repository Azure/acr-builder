// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import "testing"

// TestNewInMemoryFile tests creating an in-memory file.
func TestNewInMemoryFile(t *testing.T) {
	expectedName := "/var/logs/foo/foo.txt"
	expectedData := "bar"

	f := NewInMemoryFile(expectedName, []byte(expectedData))

	if f.Name != expectedName {
		t.Errorf("Expected %s as the file's name but received %s", expectedName, f.Name)
	}

	sData := string(f.Data)
	if sData != expectedData {
		t.Errorf("Expected %s as the file's data but received %s", expectedData, sData)
	}
}
