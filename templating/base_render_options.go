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
	// Path to the task file.
	TaskFile string

	// Base64 encoded task file.
	Base64EncodedTaskFile string

	// Path to a values file.
	ValuesFile string

	// Base64 encoded values file.
	Base64EncodedValuesFile string

	// Override values.
	TemplateValues []string

	// ID is the build ID.
	ID string

	// Commit is the commit used when running the build.
	Commit string

	// Repository is the repository used when running the build.
	Repository string

	// Branch is the branch used when running the build.
	Branch string

	// TriggeredBy is the reason the build was triggered.
	TriggeredBy string

	// GitTag is a Git tag.
	GitTag string

	// Registry is the ACR being used.
	Registry string

	// Date is the date of the Build.
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
	} else if opts.Base64EncodedValuesFile != "" {
		if config, err = DecodeConfig(opts.Base64EncodedValuesFile); err != nil {
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
		return "", errors.New("Rendered template was empty")
	}

	return rendered[template.Name], nil
}

// parseValues receives a slice of values in key=val format
// and serializes them into YAML. If a key is specified more
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
			return "", errors.New("failed to parse --set data; expected a key=val format")
		}
		val := v[i+1:] // Skip the = separator
		ret[key] = val
	}

	return ret.ToYAMLString()
}
