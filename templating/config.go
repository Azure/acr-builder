// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

// Config represents configuration values.
type Config struct {
	RawValue string            `json:"rawValue,omitempty"`
	Values   map[string]*Value `json:"values,omitempty"`
}

// GetRawValue returns the Config's value as a string.
func (c *Config) GetRawValue() string {
	if c == nil {
		return ""
	}
	return c.RawValue
}

// GetValues returns the Config's values.
func (c *Config) GetValues() map[string]*Value {
	if c == nil {
		return nil
	}
	return c.Values
}

// IsValidConfig determines whether or not the Config is valid.
func (c *Config) IsValidConfig() bool {
	return c != nil && c.RawValue != ""
}

// Value represents a configuration value.
type Value struct {
	Value string `json:"value,omitempty"`
}

// GetValue returns the Value's value.
func (v *Value) GetValue() string {
	if v == nil {
		return ""
	}
	return v.Value
}
