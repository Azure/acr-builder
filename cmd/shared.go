// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/Azure/acr-builder/templating"
)

// AddBaseRenderingOptions adds base rendering options to the specified flagset and maps the flags
// to the options struct.
func AddBaseRenderingOptions(f *flag.FlagSet, opts *templating.BaseRenderOptions, cmd *cobra.Command, usesTask bool) {

	// Templates & values files
	f.StringVar(&opts.ValuesFile, "values", "", "the values file to use")
	f.StringVar(&opts.Base64EncodedValuesFile, "encoded-values", "", "a Base64 encoded values file (overrides the values file if one is specified)")
	f.StringArrayVar(&opts.TemplateValues, "set", []string{}, "set values on the command line (use `--set` multiple times or use commas: key1=val1,key2=val2)")

	// Base rendering options
	f.StringVar(&opts.ID, "id", "", "the build ID")
	f.StringVarP(&opts.Commit, "commit", "c", "", "the commit SHA")
	f.StringVar(&opts.Repository, "repository", "", "the build repository")
	f.StringVarP(&opts.Branch, "branch", "b", "", "the build branch")
	f.StringVar(&opts.TriggeredBy, "triggered-by", "", "what the build was triggered by")
	f.StringVar(&opts.GitTag, "git-tag", "", "the git tag")
	f.StringVarP(&opts.Registry, "registry", "r", "", "the name of the registry")

	opts.Date = time.Now().UTC()

	// exec and render both use task and it's required, but build doesn't
	if usesTask {
		f.StringVar(&opts.TaskFile, "task", "", "the task file to use")
		f.StringVar(&opts.Base64EncodedTaskFile, "encoded-task", "", "a Base64 encoded task file")
	}
}
