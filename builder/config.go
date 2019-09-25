// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

// Config represents configuration values.
type Config struct {
	RawValue string
	Values   map[string]*Value
}

// GetRawValue returns the Config's value as a string.
func (c *Config) GetRawValue() string {
	if c == nil {
		return ""
	}
	return c.RawValue
}

// IsValidConfig determines whether or not the Config is valid.
func (c *Config) IsValidConfig() bool {
	return c != nil && c.RawValue != ""
}

// Value represents a configuration value.
type Value struct {
	Value string
}
