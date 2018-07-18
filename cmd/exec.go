// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/taskmanager"
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

	registryUser string
	registryPw   string

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

	f.StringVarP(&e.registryUser, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&e.registryPw, "password", "p", "", "the password to use when logging into the registry")
	f.BoolVar(&e.dryRun, "dry-run", false, "evaluates the pipeline but doesn't execute it")

	AddBaseRenderingOptions(f, e.opts, cmd, true)
	return cmd
}

func (e *execCmd) run(cmd *cobra.Command, args []string) error {
	template, err := templating.LoadTemplate(e.opts.StepsFile)
	if err != nil {
		return err
	}

	rendered, err := templating.LoadAndRenderSteps(template, e.opts)
	if err != nil {
		return err
	}

	if debug {
		fmt.Println("Rendered template:")
		fmt.Println(rendered)
	}

	pipeline, err := graph.UnmarshalPipelineFromString(rendered, e.opts.Registry, e.registryUser, e.registryPw)
	if err != nil {
		return err
	}

	if err := e.validateCmdArgs(pipeline.Push); err != nil {
		return err
	}

	taskManager := taskmanager.NewTaskManager(e.dryRun)

	timeout := time.Duration(pipeline.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
	if !e.dryRun {
		fmt.Printf("Setting up the home volume: %s\n", homeVolName)
		v := volume.NewVolume(homeVolName, taskManager)
		if msg, err := v.Create(ctx); err != nil {
			return fmt.Errorf("Err creating docker vol. Msg: %s, Err: %v", msg, err)
		}
		defer func() {
			if msg, err := v.Delete(ctx); err != nil {
				fmt.Printf("Failed to clean up docker vol: %s. Msg: %s, Err: %v\n", homeVolName, msg, err)
			}
		}()
	}

	builder := builder.NewBuilder(taskManager, debug, homeVolName)
	defer builder.CleanAllBuildSteps(context.Background(), pipeline)

	return builder.RunAllBuildSteps(ctx, pipeline)
}

func (e *execCmd) validateCmdArgs(imgs []string) error {
	if err := validateRegistryCreds(e.registryUser, e.registryPw); err != nil {
		return err
	}

	if err := validatePush(len(imgs) > 0, e.opts.Registry, e.registryUser, e.registryPw); err != nil {
		return err
	}

	return nil
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

	return ret.ToYAMLString()
}
