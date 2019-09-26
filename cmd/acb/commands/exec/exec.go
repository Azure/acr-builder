// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exec

import (
	gocontext "context"
	"fmt"
	"runtime"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/executor"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/secretmgmt"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	defaultTaskFile = "acb.yaml"
)

// Command executes a task file.
var Command = cli.Command{
	Name:  "exec",
	Usage: "execute a task file",
	Flags: []cli.Flag{
		// Task options
		cli.StringFlag{
			Name:  "file,f",
			Usage: "the path to the task file",
		},
		cli.StringFlag{
			Name:  "encoded-file",
			Usage: "a base64 encoded task file",
		},
		cli.StringFlag{
			Name:  "working-directory",
			Usage: "the default working directory to use if the underlying Task doesn't have one specified",
		},
		cli.StringFlag{
			Name:  "network",
			Usage: "the default network to use",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "the default environment variables which are applied to each step (use --env multiple times or use commas: env1=val1,env2=val2)",
		},
		cli.StringSliceFlag{
			Name:  "credential",
			Usage: "login credentials for custom registry",
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "evaluates the command, but doesn't execute it",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enables diagnostic logging",
		},

		// Rendering options
		cli.StringFlag{
			Name:  "values",
			Usage: "the path to the values file to use for rendering",
		},
		cli.StringFlag{
			Name:  "encoded-values",
			Usage: "a base64 encoded values file to use for rendering",
		},
		cli.StringFlag{
			Name:  "homevol",
			Usage: "the home volume to use",
		},
		cli.StringFlag{
			Name:  "id",
			Usage: "the unique run identifier",
		},
		cli.StringFlag{
			Name:  "commit,c",
			Usage: "the commit SHA that triggered the run",
		},
		cli.StringFlag{
			Name:  "repository",
			Usage: "the run's repository",
		},
		cli.StringFlag{
			Name:  "branch",
			Usage: "the git branch",
		},
		cli.StringFlag{
			Name:  "triggered-by",
			Usage: "describes what the run was triggered by",
		},
		cli.StringFlag{
			Name:  "git-tag",
			Usage: "the git tag that triggered the run",
		},
		cli.StringFlag{
			Name:  "registry,r",
			Usage: "the fully qualified name of the registry",
		},
		cli.StringFlag{
			Name:  "os-version",
			Usage: "the version of the OS",
		},
		cli.StringSliceFlag{
			Name:  "set",
			Usage: "set values on the command line (use --set multiple times or use commas: key1=val1,key2=val2)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "the name of the task",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			// Task options
			taskFile                = context.String("file")
			encodedTaskFile         = context.String("encoded-file")
			defaultWorkingDirectory = context.String("working-directory")
			defaultNetwork          = context.String("network")
			defaultEnvs             = context.StringSlice("env")
			creds                   = context.StringSlice("credential")
			dryRun                  = context.Bool("dry-run")
			debug                   = context.Bool("debug")

			// Rendering options
			values        = context.String("values")
			encodedValues = context.String("encoded-values")
			homevol       = context.String("homevol")
			id            = context.String("id")
			commit        = context.String("commit")
			repository    = context.String("repository")
			branch        = context.String("branch")
			triggeredBy   = context.String("triggered-by")
			tag           = context.String("git-tag")
			registry      = context.String("registry")
			osVersion     = context.String("os-version")
			setVals       = context.StringSlice("set")
			taskName      = context.String("name")
		)

		if taskFile == "" && encodedTaskFile == "" {
			taskFile = defaultTaskFile
		}

		ctx := gocontext.WithValue(gocontext.Background(), "debug", debug)
		pm := procmanager.NewProcManager(dryRun)

		if homevol == "" {
			if !dryRun {
				homevol = fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
				v := volume.NewVolume(homevol, pm)
				if msg, err := v.Create(ctx); err != nil {
					return fmt.Errorf("failed to create volume. Msg: %s, Err: %v", msg, err)
				}
				defer func() {
					_, _ = v.Delete(ctx)
				}()
			}
		}

		task, err := builder.CreateExecTask(ctx, &builder.TaskCreateOptions{
			TaskFile:                taskFile,
			Base64EncodedTaskFile:   encodedTaskFile,
			WorkingDirectory:        defaultWorkingDirectory,
			Network:                 defaultNetwork,
			Env:                     defaultEnvs,
			Credentials:             creds,
			ValuesFile:              values,
			Base64EncodedValuesFile: encodedValues,
			SharedVolume:            homevol,
			ID:                      id,
			Commit:                  commit,
			Repository:              repository,
			Branch:                  branch,
			TriggeredBy:             triggeredBy,
			GitTag:                  tag,
			Registry:                registry,
			Date:                    time.Now().UTC(),
			OS:                      runtime.GOOS,
			OSVersion:               osVersion,
			Architecture:            runtime.GOARCH,
			SecretResolveTimeout:    secretmgmt.DefaultSecretResolveTimeout,
			TemplateValues:          setVals,
			TaskName:                taskName,
		})

		if err != nil {
			return errors.Wrap(err, "failed to build a task")
		}

		executor := executor.NewExecutor(pm, debug, homevol)
		defer executor.CleanTask(gocontext.Background(), task) // Use a separate context since the other may have expired.
		return executor.RunTask(gocontext.Background(), task)
	},
}
