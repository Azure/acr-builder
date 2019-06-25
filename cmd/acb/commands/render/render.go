// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package render

import (
	gocontext "context"
	"errors"
	"log"
	"runtime"
	"time"

	"github.com/Azure/acr-builder/secretmgmt"
	"github.com/Azure/acr-builder/templating"
	"github.com/urfave/cli"
)

// Command renders templates and verifies their output.
var Command = cli.Command{
	Name:  "render",
	Usage: "render the specified template",
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
			// Task options
			taskFile        = context.String("file")
			encodedTaskFile = context.String("encoded-file")

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

			renderOpts = &templating.BaseRenderOptions{
				TaskFile:                taskFile,
				Base64EncodedTaskFile:   encodedTaskFile,
				ValuesFile:              values,
				Base64EncodedValuesFile: encodedValues,
				TemplateValues:          setVals,
				ID:                      id,
				Commit:                  commit,
				Repository:              repository,
				Branch:                  branch,
				TriggeredBy:             triggeredBy,
				GitTag:                  tag,
				Registry:                registry,
				Date:                    time.Now().UTC(),
				SharedVolume:            homevol,
				OS:                      runtime.GOOS,
				OSVersion:               osVersion,
				Architecture:            runtime.GOARCH,
				SecretResolveTimeout:    secretmgmt.DefaultSecretResolveTimeout,
			}
		)

		if taskFile == "" && encodedTaskFile == "" {
			return errors.New("a task file or base64 encoded task file is required")
		}

		var template *templating.Template
		var err error
		if taskFile == "" {
			if template, err = templating.DecodeTemplate(encodedTaskFile); err != nil {
				return err
			}
		} else {
			if template, err = templating.LoadTemplate(taskFile); err != nil {
				return err
			}
		}

		rendered, err := templating.LoadAndRenderSteps(gocontext.Background(), template, renderOpts)
		if err != nil {
			return err
		}

		log.Println("Rendered template:")
		log.Println(rendered)
		return nil
	},
}
