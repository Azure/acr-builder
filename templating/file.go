// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

// InMemoryFile represents an in-memory file.
type InMemoryFile struct {
	Name string
	Data []byte
}

// NewInMemoryFile makes a new InMemoryFile.
func NewInMemoryFile(name string, data []byte) *InMemoryFile {
	return &InMemoryFile{
		Name: name,
		Data: data,
	}
}
