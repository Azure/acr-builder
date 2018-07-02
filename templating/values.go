// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Azure/acr-builder/util"
	"github.com/BurntSushi/toml"
)

// BaseRenderOptions represents additional information for the composition of the final rendering.
type BaseRenderOptions struct {
	// ID is the build ID. Required.
	ID string
	// Commit is the commit used when running the build. Optional.
	Commit string
	// Tag is the tag used when running the build. Optional.
	Tag string
	// Repository is the repository used when running the build. Optional.
	Repository string
	// Branch is the branch used when running the build. Optional.
	Branch string
	// TriggeredBy is the reason the build was triggered. Required.
	TriggeredBy string
	// Registry is the ACR being used.
	Registry string
}

// Values represents a map of build values.
type Values map[string]interface{}

// ToTOMLString encodes the Values object into a TOML string.
func (v Values) ToTOMLString() (string, error) {
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(v)
	return buf.String(), err
}

// Deserialize will convert the specified bytes to a Values object.
func Deserialize(b []byte) (v Values, err error) {
	v = Values{}
	_, err = toml.Decode(string(b), &v)
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

// OverrideValuesWithBuildInfo overrides the specified job's values and provides a default set of values.
func OverrideValuesWithBuildInfo(j *Job, c *Config, options BaseRenderOptions) (Values, error) {
	base := map[string]interface{}{
		"Build": map[string]interface{}{
			"ID":          options.ID,
			"Commit":      options.Commit,
			"Tag":         options.Tag,
			"Repository":  options.Repository,
			"Branch":      options.Branch,
			"TriggeredBy": options.TriggeredBy,
			"Registry":    options.Registry,
		},
	}

	vals, err := OverrideValues(j, c)
	if err != nil {
		return base, err
	}

	base["Values"] = vals
	return base, nil
}

// OverrideValues overrides the Values of a job.
func OverrideValues(j *Job, c *Config) (Values, error) {
	merged := Values{}

	if c != nil {
		v, err := Deserialize([]byte(c.RawValue))
		if err != nil {
			return merged, err
		}

		merged, err = merge(j, v)
		if err != nil {
			return merged, err
		}
	}

	return merged, nil
}

// merge merges the specified job with the specified map.
// The specified map has precendence over the job.
func merge(j *Job, merged map[string]interface{}) (map[string]interface{}, error) {
	if !j.HasValidConfig() {
		return merged, nil
	}

	curr, err := Deserialize([]byte(j.Config.RawValue))
	if err != nil {
		return merged, fmt.Errorf("Failed to deserialize values during merge: %s, Err: %v", j.Config.RawValue, err)
	}

	for k, v := range curr {
		if currVal, ok := merged[k]; ok {

			// If the lookup is nil, remove the key.
			// This allows us to remove keys during overrides if they no longer exist.
			// I.e., someone broke compatibility in a future template.
			if currVal == nil {
				delete(merged, k)
			} else if sink, ok := currVal.(map[string]interface{}); ok {
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
			// 3. Otherwise, the key is trying to be overriden by a scalar value, in which
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
