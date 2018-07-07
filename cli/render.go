// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cli

import (
	"fmt"
	"io"

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
	AddBaseRenderingOptions(f, r.opts, cmd)
	return cmd
}

func (r *renderCmd) run(cmd *cobra.Command, args []string) error {
	template, err := templating.LoadAndRenderSteps(r.opts)
	if err != nil {
		return err
	}

	fmt.Println("Rendered template:")
	fmt.Println(template)
	return nil
}
