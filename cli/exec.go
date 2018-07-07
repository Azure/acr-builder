// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/cmder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/templating"
	"github.com/Azure/acr-builder/volume"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const execLongDesc = `
This command can be used to execute a pipeline.
`

type execCmd struct {
	out    io.Writer
	dryRun bool

	registryUserName string
	registryPassword string

	opts *templating.BaseRenderOptions
}

func newExecCmd(out io.Writer) *cobra.Command {
	e := &execCmd{
		out:  out,
		opts: &templating.BaseRenderOptions{},
	}

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a pipeline",
		Long:  execLongDesc,
		RunE:  e.run,
	}

	f := cmd.Flags()

	f.StringVarP(&e.registryUserName, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&e.registryPassword, "password", "p", "", "the password to use when logging into the registry")
	f.BoolVar(&e.dryRun, "dry-run", false, "evaluates the pipeline but doesn't execute it")

	AddBaseRenderingOptions(f, e.opts, cmd)
	return cmd
}

func (e *execCmd) run(cmd *cobra.Command, args []string) error {
	template, err := templating.LoadAndRenderSteps(e.opts)
	if err != nil {
		return err
	}
	if template == "" {
		return errors.New("rendered pipeline was empty")
	}

	if debug {
		fmt.Println("Rendered template:")
		fmt.Println(template)
	}

	p, err := graph.UnmarshalPipelineFromString(template)
	if err != nil {
		return err
	}

	dag, err := graph.NewDagFromPipeline(p)
	if err != nil {
		return err
	}

	cmder := cmder.NewCmder(e.dryRun)

	timeout := time.Duration(p.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
	if !e.dryRun {
		fmt.Printf("Setting up the home volume: %s\n", homeVolName)
		v := volume.NewVolume(homeVolName, cmder)
		if msg, err := v.Create(ctx); err != nil {
			return fmt.Errorf("Err creating docker vol. Msg: %s, Err: %v", msg, err)
		}
		defer func() {
			if msg, err := v.Delete(ctx); err != nil {
				fmt.Printf("Failed to clean up docker vol: %s. Msg: %s, Err: %v\n", homeVolName, msg, err)
			}
		}()
	}

	buildOptions := &builder.BuildOptions{
		RegistryName:     e.opts.Registry,
		RegistryUsername: e.registryUserName,
		RegistryPassword: e.registryPassword,
		Push:             len(p.Push) > 0,
	}

	builder := builder.NewBuilder(cmder, debug, homeVolName, buildOptions)
	defer builder.CleanAllBuildSteps(context.Background(), dag)

	return builder.RunAllBuildSteps(ctx, dag, p.Push)
}

func combineVals(values []string) (string, error) {
	ret := templating.Values{}
	for _, v := range values {
		s := strings.Split(v, "=")
		if len(s) != 2 {
			return "", fmt.Errorf("failed to parse --set data: %s", v)
		}
		ret[s[0]] = s[1]
	}

	return ret.ToTOMLString()
}
