// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Azure/acr-builder/util"
	yaml "gopkg.in/yaml.v2"
)

// Values represents a map of build values.
type Values map[string]interface{}

// ToYAMLString encodes the Values object into a YAML string.
func (v Values) ToYAMLString() (string, error) {
	b, err := yaml.Marshal(v)
	return string(b), err
}

// Deserialize will convert the specified bytes to a Values object.
func Deserialize(b []byte) (v Values, err error) {
	v = Values{}
	err = yaml.Unmarshal(b, &v)
	if len(v) == 0 {
		v = Values{}
	}
	return v, err
}

// DeserializeFromFile will parse the specified file name and convert it
// to a Values object.
func DeserializeFromFile(fileName string) (Values, error) {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return Deserialize(b)
}

// OverrideValues overrides the first config with the second.
func OverrideValues(c1 *Config, c2 *Config) (Values, error) {
	merged := Values{}

	if c2 != nil {
		v, err := Deserialize([]byte(c2.RawValue))
		if err != nil {
			return merged, err
		}

		merged, err = merge(c1, v)
		if err != nil {
			return merged, err
		}
	}

	return merged, nil
}

// merge merges the specified template with the specified map.
// The specified map has precedence.
func merge(c *Config, merged map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsValidConfig() {
		return merged, nil
	}

	vals, err := Deserialize([]byte(c.RawValue))
	if err != nil {
		return merged, fmt.Errorf("Failed to deserialize values. Try linting your template locally. Err: %v", err)
	}

	for k, v := range vals {
		if lookup, ok := merged[k]; ok {

			// If the lookup is nil, remove the key.
			// This allows us to remove keys during overrides if they no longer exist.
			// I.e., someone broke compatibility in a future template.
			if lookup == nil {
				delete(merged, k)
			} else if sink, ok := lookup.(map[string]interface{}); ok {
				source, ok := v.(map[string]interface{})
				if !ok {
					log.Printf("Skip merging: %s. Not a map", k)
					continue
				}

				// The to-be-merged value has precedence over the start value.
				mergeMaps(sink, source)
			}
		} else {
			// If the key doesn't exist, copy it.
			merged[k] = v
		}
	}
	return merged, nil
}

// mergeMaps merges two maps.
func mergeMaps(sink, source map[string]interface{}) map[string]interface{} {
	for k, v := range source {

		// Try to pull out a map from the source
		if _, ok := v.(map[string]interface{}); ok {

			// 1. If the key doesn't exist, set it.
			// 2. If the key exists and it's a map, recursively iterate through the map.
			// 3. Otherwise, the key is trying to be overridden by a scalar value, in which
			// case print a warning message and skip it.
			if innerV, ok := sink[k]; !ok {
				sink[k] = v
			} else if util.IsMap(innerV) {
				mergeMaps(innerV.(map[string]interface{}), v.(map[string]interface{}))
			} else {
				log.Printf("Skip merging: %s. Can't override a map with a scalar %v", k, v)
			}
		} else {
			sv, ok := sink[k]
			if ok && util.IsMap(sv) {
				log.Printf("Skip merging: %s is a map but %v is not", k, v)
			} else {
				sink[k] = v
			}
		}
	}

	return sink
}
