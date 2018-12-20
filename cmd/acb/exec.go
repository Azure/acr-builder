// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/templating"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	execUse = "exec"

	execShortDesc = "Execute a task"

	execLongDesc = `
This command can be used to execute a task.
`

	defaultTaskFile = "acb.yaml"
)

type execCmd struct {
	out    io.Writer
	dryRun bool

	registryUser   string
	registryPw     string
	credentials    []string
	defaultWorkDir string
	network        string
	envs           []string

	opts *templating.BaseRenderOptions
}

func newExecCmd(out io.Writer) *cobra.Command {
	e := &execCmd{
		out:  out,
		opts: &templating.BaseRenderOptions{},
	}

	cmd := &cobra.Command{
		Use:   execUse,
		Short: execShortDesc,
		Long:  execLongDesc,
		RunE:  e.run,
	}

	f := cmd.Flags()

	f.StringVarP(&e.registryUser, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&e.registryPw, "password", "p", "", "the password to use when logging into the registry")
	f.StringArrayVar(&e.credentials, "credentials", []string{}, "all credentials for private repos")

	f.BoolVar(&e.dryRun, "dry-run", false, "evaluates the task but doesn't execute it")
	f.StringVar(&e.defaultWorkDir, "working-directory", "", "the default working directory to use if the underlying Task doesn't have one specified")

	f.StringVar(&e.network, "network", "", "the default network to use")
	f.StringArrayVar(&e.envs, "env", []string{}, "the default environment variables which are applied to each step (use `--env` multiple times or use commas: env1=val1,env2=val2)")

	AddBaseRenderingOptions(f, e.opts, cmd, true)
	return cmd
}

func (e *execCmd) run(cmd *cobra.Command, args []string) error {
	e.setDefaultTaskFile()

	ctx := context.Background()

	procManager := procmanager.NewProcManager(e.dryRun)
	if e.opts.SharedVolume == "" {
		if !e.dryRun {
			homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
			e.opts.SharedVolume = homeVolName
			v := volume.NewVolume(homeVolName, procManager)
			if msg, err := v.Create(ctx); err != nil {
				return fmt.Errorf("Err creating docker vol. Msg: %s, Err: %v", msg, err)
			}
			defer func() {
				_, _ = v.Delete(ctx)
			}()
		}
	}
	log.Printf("Using %s as the home volume\n", e.opts.SharedVolume)

	var template *templating.Template
	var err error
	if e.opts.TaskFile == "" {
		if template, err = templating.DecodeTemplate(e.opts.Base64EncodedTaskFile); err != nil {
			return err
		}
	} else {
		if template, err = templating.LoadTemplate(e.opts.TaskFile); err != nil {
			return err
		}
	}

	rendered, err := templating.LoadAndRenderSteps(template, e.opts)
	if err != nil {
		return err
	}

	if debug {
		log.Println("Rendered template:")
		log.Println(rendered)
	}

	var credentials []*graph.Credential
	// If the user provides the username and password, add it to the Credentials
	if e.opts.Registry != "" && e.registryUser != "" && e.registryPw != "" {
		cred, err := graph.NewCredential(e.opts.Registry, e.registryUser, e.registryPw)
		if err != nil {
			return err
		}
		credentials = append(credentials, cred)
	}

	// Add any additional creds provided by the user in the --credentials flag
	for _, credString := range e.credentials {
		// creds should be of the form of "regName;userName;password". If not, return an error
		cred, err := graph.CreateCredentialFromString(credString)
		if err != nil {
			return err
		}

		credentials = append(credentials, cred)
	}

	task, err := graph.UnmarshalTaskFromString(rendered, e.defaultWorkDir, e.network, e.envs, credentials)

	if err != nil {
		return err
	}

	if err := e.validateCmdArgs(); err != nil {
		return err
	}

	timeout := time.Duration(task.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	builder := builder.NewBuilder(procManager, debug, e.opts.SharedVolume)
	defer builder.CleanTask(context.Background(), task)
	return builder.RunTask(ctx, task)
}

func (e *execCmd) validateCmdArgs() error {
	return validateRegistryCreds(e.registryUser, e.registryPw)
}

func (e *execCmd) setDefaultTaskFile() {
	if e.opts.TaskFile == "" && e.opts.Base64EncodedTaskFile == "" {
		e.opts.TaskFile = defaultTaskFile
	}
}
