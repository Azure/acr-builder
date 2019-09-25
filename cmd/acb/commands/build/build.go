// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package build

import (
	gocontext "context"
	"fmt"
	"log"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/executor"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Command executes a container build.
var Command = cli.Command{
	Name:      "build",
	Usage:     "build container images",
	ArgsUsage: "[path|url]",
	Flags: []cli.Flag{
		// Build options
		cli.StringFlag{
			Name:  "file,f",
			Usage: "the path to the Dockerfile",
			Value: "Dockerfile",
		},
		cli.StringFlag{
			Name:  "working-directory",
			Usage: "the default working directory to use",
		},
		cli.StringFlag{
			Name:  "target",
			Usage: "specifies the target stage to build in a multi-stage build",
		},
		cli.StringFlag{
			Name:  "isolation",
			Usage: "build isolation",
		},
		cli.StringFlag{
			Name:  "platform",
			Usage: "sets the platform if the server is capable of multiple platforms",
		},
		cli.StringSliceFlag{
			Name:  "tag,t",
			Usage: "name and optionally a tag in 'name:tag' format",
		},
		cli.StringSliceFlag{
			Name:  "build-arg",
			Usage: "build arguments",
		},
		cli.StringSliceFlag{
			Name:  "secret-build-arg",
			Usage: "secret build arguments",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "set metadata for an image",
		},
		cli.StringSliceFlag{
			Name:  "credential",
			Usage: "login credentials for custom registry",
		},
		cli.BoolFlag{
			Name:  "pull",
			Usage: "attempt to pull a newer version of the base image during build",
		},
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "ignore all cached layers when building an image",
		},
		cli.BoolFlag{
			Name:  "push",
			Usage: "push the image on success",
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
	},
	Action: func(context *cli.Context) error {
		var (
			// Build options
			buildContext            = context.Args().First()
			dockerfile              = context.String("file")
			defaultWorkingDirectory = context.String("working-directory")
			target                  = context.String("target")
			isolation               = context.String("isolation")
			platform                = context.String("platform")
			tags                    = context.StringSlice("tag")
			buildArgs               = context.StringSlice("build-arg")
			secretBuildArgs         = context.StringSlice("secret-build-arg")
			labels                  = context.StringSlice("label")
			creds                   = context.StringSlice("credential")
			pull                    = context.Bool("pull")
			noCache                 = context.Bool("no-cache")
			push                    = context.Bool("push")
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
		)

		if buildContext == "" {
			return errors.New("build requires exactly 1 argument, see build --help")
		}
		if err := validateIsolation(isolation); err != nil {
			return err
		}
		if err := validatePush(push, creds); err != nil {
			return err
		}

		ctx := gocontext.Background()
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
		log.Printf("Using %s as the home volume\n", homevol)

		task, err := builder.CreateBuildTask(ctx, &builder.TaskCreateOptions{
			BuildContext:            buildContext,
			Dockerfile:              dockerfile,
			WorkingDirectory:        defaultWorkingDirectory,
			Target:                  target,
			Isolation:               isolation,
			Platform:                platform,
			Tags:                    tags,
			BuildArgs:               buildArgs,
			SecretBuildArgs:         secretBuildArgs,
			Labels:                  labels,
			Pull:                    pull,
			NoCache:                 noCache,
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
			OSVersion:               osVersion,
			TemplateValues:          setVals,
			Credentials:             creds,
		})
		if err != nil {
			return errors.Wrap(err, "failed to build a task")
		}

		executor := executor.NewBuilder(pm, debug, homevol)
		defer executor.CleanTask(gocontext.Background(), task) // Use a separate context since the other may have expired.
		return executor.RunTask(gocontext.Background(), task)
	},
}
