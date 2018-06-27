// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
)

const initLongDesc = `
This command can be used to initialize a new template.
`

type initCmd struct {
	out io.Writer
}

func newInitCmd(out io.Writer) *cobra.Command {
	i := &initCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a default template",
		Long:  initLongDesc,
		RunE:  i.run,
	}

	return cmd
}

func (i *initCmd) run(cmd *cobra.Command, args []string) error {
	// TODO: implement
	return errors.New("init isn't implemented yet")
}
