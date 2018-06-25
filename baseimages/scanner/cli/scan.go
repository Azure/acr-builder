package cli

import (
	"context"
	"encoding/json"

	"fmt"
	"io"
	"time"

	"github.com/Azure/acr-builder/baseimages/scanner/scan"
	"github.com/Azure/acr-builder/cmder"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const scanLongDesc = `
This command can be used to scan a Dockerifle.
`

type scanCmd struct {
	out               io.Writer
	dockerfile        string
	context           string
	tags              []string
	buildArgs         []string
	timeout           int
	destinationFolder string
}

func newScanCmd(out io.Writer) *cobra.Command {
	s := &scanCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "scan [OPTIONS] PATH | URL",
		Short: "Scan a Dockerfile",
		Long:  scanLongDesc,
		RunE:  s.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("scanner scan requires exactly 1 argument. See scanner scan --help")
			}

			return nil
		},
	}

	f := cmd.Flags()

	f.StringVarP(&s.dockerfile, "file", "f", "Dockerfile", "path to the Dockerfile")
	f.StringArrayVarP(&s.tags, "tag", "t", []string{}, "name and optionally a tag in the 'name:tag' format")
	f.StringArrayVar(&s.buildArgs, "build-arg", []string{}, "set build time arguments")
	f.IntVar(&s.timeout, "timeout", 60, "maximum execution time")
	f.StringVar(&s.destinationFolder, "destination", "temp", "the destination folder to save context")
	return cmd
}

func (s *scanCmd) run(cmd *cobra.Command, args []string) error {
	s.context = args[0]
	timeout := time.Duration(s.timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmder := cmder.NewCmder(false)

	scanner := scan.NewScanner(cmder, s.context, s.dockerfile, s.destinationFolder, s.buildArgs, s.tags, debug)
	deps, err := scanner.Scan(ctx)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(deps)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal image dependencies")
	}

	fmt.Println(string(bytes))
	return nil
}
