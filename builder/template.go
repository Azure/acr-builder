// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"strings"
)

// Template represents a template.
type Template struct {
	Name string
	Data []byte
}

// GetName returns a Template's name.
func (t *Template) GetName() string {
	if t == nil {
		return ""
	}
	return t.Name
}

// GetData returns a Template's data.
func (t *Template) GetData() []byte {
	if t == nil {
		return nil
	}
	return t.Data
}

// NewTemplate creates a new template with the specified name and data.
// It will strip out any commented lines from data, i.e. lines beginning with #.
func NewTemplate(name string, data []byte) *Template {
	ret := []string{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		tLine := strings.TrimSpace(line)
		if !strings.HasPrefix(tLine, "#") {
			// Append the original line to preserve any spacing.
			ret = append(ret, line)
		}
	}
	return &Template{
		Name: name,
		Data: []byte(strings.Join(ret, "\n")),
	}
}
