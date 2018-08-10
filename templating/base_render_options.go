// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// BaseRenderOptions represents additional information for the composition of the final rendering.
type BaseRenderOptions struct {
	// Path to the steps file. Required.
	StepsFile string

	// Path to a values file. Optional.
	ValuesFile string

	// Override values. Optional.
	TemplateValues []string

	// ID is the build ID. Required.
	ID string

	// Commit is the commit used when running the build. Optional.
	Commit string

	// Repository is the repository used when running the build. Optional.
	Repository string

	// Branch is the branch used when running the build. Optional.
	Branch string

	// TriggeredBy is the reason the build was triggered. Required.
	TriggeredBy string

	// GitTag is a Git tag. Optional.
	GitTag string

	// Registry is the ACR being used. Optional.
	Registry string

	// Date is the date of the Build. Required.
	Date time.Time
}

// OverrideValuesWithBuildInfo overrides the specified config's values and provides a default set of values.
func OverrideValuesWithBuildInfo(c1 *Config, c2 *Config, options *BaseRenderOptions) (Values, error) {
	base := map[string]interface{}{
		"Build": map[string]interface{}{
			"ID": options.ID,
		},
		"Run": map[string]interface{}{
			"ID":          options.ID,
			"Commit":      options.Commit,
			"Repository":  options.Repository,
			"Branch":      options.Branch,
			"GitTag":      options.GitTag,
			"TriggeredBy": options.TriggeredBy,
			"Registry":    options.Registry,
			"Date":        options.Date.Format("20060102-150405z"), // yyyyMMdd-HHmmssz
		},
	}

	vals, err := OverrideValues(c1, c2)
	if err != nil {
		return base, err
	}

	base["Values"] = vals
	return base, nil
}

// LoadAndRenderSteps loads a template file and renders it according to an optional values file, --set values,
// and base render options.
func LoadAndRenderSteps(template *Template, opts *BaseRenderOptions) (string, error) {
	var err error

	config := &Config{}
	if opts.ValuesFile != "" {
		if config, err = LoadConfig(opts.ValuesFile); err != nil {
			return "", err
		}
	}

	setConfig := &Config{}
	if len(opts.TemplateValues) > 0 {
		rawVals, err := parseValues(opts.TemplateValues)
		if err != nil {
			return "", err
		}

		setConfig = &Config{RawValue: rawVals, Values: map[string]*Value{}}
	}

	mergedVals, err := OverrideValuesWithBuildInfo(config, setConfig, opts)
	if err != nil {
		return "", fmt.Errorf("Failed to override values: %v", err)
	}

	engine := NewEngine()
	rendered, err := engine.Render(template, mergedVals)
	if err != nil {
		return "", fmt.Errorf("Error while rendering templates: %v", err)
	}

	if rendered[template.Name] == "" {
		return "", fmt.Errorf("Rendered template was empty. Original template: %s", template.Data)
	}

	return rendered[template.Name], nil
}

// parseValues receives a slice of values in key=val format
// and serializes them into JSON. If a key is specified more
// than once, the key will be overridden.
func parseValues(values []string) (string, error) {
	ret := Values{}
	for _, v := range values {
		i := strings.Index(v, "=")
		if i < 0 {
			return "", errors.New("failed to parse --set data; invalid format, no = assignment found")
		}
		key := v[:i]
		if key == "" {
			return "", errors.New("failed to parse --set data; invalid format, empty key")
		}
		val := v[i+1:] // Skip the = separator
		ret[key] = val
	}

	return ret.ToYAMLString()
}
