// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/secretmgmt"
	"github.com/pkg/errors"
)

var shellEscapePattern = regexp.MustCompile(`[^\w_^@=+%,:./-]`)

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

	// ID is a unique identifier for the run.
	ID string

	// Commit is the SHA the run was triggered against.
	Commit string

	// Repository is the repository the run was triggered against.
	Repository string

	// Branch is the branch the run was triggered against.
	Branch string

	// TriggeredBy is the reason the run was triggered.
	TriggeredBy string

	// GitTag is the git tag the run was triggered against.
	GitTag string

	// Registry is the container registry being used.
	Registry string

	// Date is the UTC date of the run.
	Date time.Time

	// SharedVolume is the name of the shared volume.
	SharedVolume string

	// OS is the GOOS.
	OS string

	// OSVersion is a specific version of the OS.
	// For example, on Windows it could be 1903.
	OSVersion string

	// Architecture is the GOARCH.
	Architecture string

	// SecretResolveTimeout is the timeout for resolving a secret during rendering.
	SecretResolveTimeout time.Duration

	// TaskName is the name of the Task executing this run
	TaskName string
}

// OverrideValuesWithBuildInfo overrides the specified config's values and provides a default set of values.
func OverrideValuesWithBuildInfo(c1 *Config, c2 *Config, opts *BaseRenderOptions) (Values, error) {
	base := map[string]interface{}{
		"Build": map[string]interface{}{
			"ID": opts.ID,
		},
		"Run": map[string]interface{}{
			"ID":           opts.ID,
			"Commit":       opts.Commit,
			"Repository":   opts.Repository,
			"Branch":       opts.Branch,
			"GitTag":       opts.GitTag,
			"TriggeredBy":  opts.TriggeredBy,
			"Registry":     opts.Registry,
			"RegistryName": parseRegistryName(opts.Registry),
			"Date":         opts.Date.Format("20060102-150405z"), // yyyyMMdd-HHmmssz
			"SharedVolume": opts.SharedVolume,
			"OS":           opts.OS,
			"OSVersion":    opts.OSVersion,
			"Architecture": opts.Architecture,
			"TaskName":     opts.TaskName,
		},
	}

	vals, err := OverrideValues(c1, c2)
	if err != nil {
		return base, err
	}

	valsJSON, err := json.Marshal(vals)
	if err != nil {
		return base, errors.Wrap(err, "failed to serialize Values")
	}
	runJSON, err := json.Marshal(base["Run"])
	if err != nil {
		return base, errors.Wrap(err, "failed to serialize Run")
	}

	base["Values"] = vals
	base["ValuesJSON"] = shellQuote(string(valsJSON))
	base["RunJSON"] = shellQuote(string(runJSON))
	return base, nil
}

// LoadAndRenderSteps loads a template file and renders it according to an optional values file, --set values,
// and base render options.
func LoadAndRenderSteps(ctx context.Context, template *Template, opts *BaseRenderOptions) (string, error) {
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
		var rawVals string
		rawVals, err = parseValues(opts.TemplateValues)
		if err != nil {
			return "", err
		}

		setConfig = &Config{RawValue: rawVals, Values: map[string]*Value{}}
	}

	mergedVals, err := OverrideValuesWithBuildInfo(config, setConfig, opts)
	if err != nil {
		return "", fmt.Errorf("failed to override values: %v", err)
	}

	engine := NewEngine()
	// we will pass nil for the secret resolve override so as to use the default resolve function.
	secrets, err := renderAndResolveSecrets(ctx, template, engine, nil, opts, mergedVals)
	if err != nil {
		return "", fmt.Errorf("failed to resolve secrets in the task with error: %v", err)
	}
	// update the secrets collection with resolved secrets.
	mergedVals["Secrets"] = secrets

	rendered, err := engine.Render(template, mergedVals)
	if err != nil {
		return "", fmt.Errorf("error while rendering templates: %v", err)
	}

	if rendered == "" {
		return "", errors.New("rendered template was empty")
	}

	return rendered, nil
}

// renderAndResolveSecrets parses the secrets in the template, resolves them using vault providers and returns the resolved secret values.
func renderAndResolveSecrets(
	ctx context.Context,
	template *Template,
	templateEngine *Engine,
	resolveSecretFunc secretmgmt.ResolveSecretFunc,
	opts *BaseRenderOptions,
	sourceValues Values) (Values, error) {
	result := Values{}
	// Cheap optimization to skip the secrets merging if it doesn't contain "secrets" string in it. Note that the task can
	// have the string secrets but may not essentially the secrets section.
	if !strings.Contains(string(template.Data), "secrets") {
		return result, nil
	}

	// At first render the template with existing values to render templatized values for secrets.
	sourceValues["Secrets"] = result
	rendered, err := templateEngine.Render(template, sourceValues)
	if err != nil {
		return result, errors.Wrap(err, "failed to render the template")
	}

	if rendered == "" {
		return result, errors.New("rendered template was empty")
	}

	// Unmarshall the template to Task and get all secrets defined in the template.
	task, err := graph.NewTaskFromString(rendered)
	if err != nil {
		return result, errors.Wrap(err, "failed to parse template to create task")
	}

	// If no secrets found return.
	if len(task.Secrets) == 0 {
		return result, nil
	}

	secretResolver, err := secretmgmt.NewSecretResolver(resolveSecretFunc, opts.SecretResolveTimeout)
	if err != nil {
		return result, errors.Wrap(err, "failed to create secret resolver")
	}

	err = secretResolver.ResolveSecrets(ctx, task.Secrets)
	if err != nil {
		return result, err
	}
	for _, s := range task.Secrets {
		result[s.ID] = s.ResolvedValue
	}
	return result, nil
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

// parseRegistryName parses the fully qualified registry name and extracts only the registry name.
// NB: This function is currently designed and provided for Azure Container Registry and may not
// work as expected for all other registries' formats.
func parseRegistryName(fullyQualifiedRegistryName string) string {
	idx := strings.Index(fullyQualifiedRegistryName, ".")
	if idx < 0 {
		return fullyQualifiedRegistryName
	}
	return fullyQualifiedRegistryName[:idx]
}

// shellQuote detects whether or not the string needs to be escaped and escapes double quotes and wraps them
// with single quotes if necessary.
func shellQuote(str string) string {
	if str == "" {
		return "''"
	}
	if shellEscapePattern.MatchString(str) {
		return "'" + strings.Replace(str, "'", "'\"'\"'", -1) + "'"
	}
	return str
}
