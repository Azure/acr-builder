// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"errors"
	"io"
	"log"

	"github.com/Azure/acr-builder/templating"
	"github.com/spf13/cobra"
)

const renderLongDesc = `
This command can be used to render your templates locally and verify their output.
`

type renderCmd struct {
	out  io.Writer
	opts *templating.BaseRenderOptions
}

func newRenderCmd(out io.Writer) *cobra.Command {
	r := &renderCmd{
		out:  out,
		opts: &templating.BaseRenderOptions{},
	}

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a template",
		Long:  renderLongDesc,
		RunE:  r.run,
	}

	f := cmd.Flags()
	AddBaseRenderingOptions(f, r.opts, cmd, true)
	return cmd
}

func (r *renderCmd) run(cmd *cobra.Command, args []string) error {
	if r.opts.TaskFile == "" && r.opts.Base64EncodedTaskFile == "" {
		return errors.New("A task file or Base64 encoded task file is required")
	}

	var template *templating.Template
	var err error
	if r.opts.TaskFile == "" {
		if template, err = templating.DecodeTemplate(r.opts.Base64EncodedTaskFile); err != nil {
			return err
		}
	} else {
		if template, err = templating.LoadTemplate(r.opts.TaskFile); err != nil {
			return err
		}
	}

	rendered, err := templating.LoadAndRenderSteps(template, r.opts)
	if err != nil {
		return err
	}

	log.Println("Rendered template:")
	log.Println(rendered)
	return nil
}
