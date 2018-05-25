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
