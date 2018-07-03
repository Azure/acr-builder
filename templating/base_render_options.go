// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

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
