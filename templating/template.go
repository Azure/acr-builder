// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

// Template represents a template.
type Template struct {
	Name string `json:"name,omitempty"`
	Data []byte `json:"data,omitempty"`
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
