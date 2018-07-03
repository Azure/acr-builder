// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
)

const lintLongDesc = `
This command can be used to lint your templates.
`

type lintCmd struct {
	out              io.Writer
	template         string
	buildID          string
	buildCommit      string
	buildTag         string
	buildRepository  string
	buildBranch      string
	buildTriggeredBy string
}

func newLintCmd(out io.Writer) *cobra.Command {
	l := &lintCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint a template",
		Long:  lintLongDesc,
		RunE:  l.run,
	}

	return cmd
}

func (b *lintCmd) run(cmd *cobra.Command, args []string) error {
	// TODO: implement
	return errors.New("linting isn't implemented yet")
}
