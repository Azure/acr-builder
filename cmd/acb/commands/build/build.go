// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package build

import (
	gocontext "context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/pkg/volume"
	"github.com/Azure/acr-builder/templating"
	"github.com/Azure/acr-builder/util"
	"github.com/google/uuid"
	"github.com/urfave/cli"
)

const (
	taskTotalTimeoutInSec = 60 * 60 * 9 // 9 hours
	buildTimeoutInSec     = 60 * 60 * 8 // 8 hours
	pushTimeoutInSec      = 60 * 30     // 30 minutes
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
			Usage: "registry credentials in the format of 'server;username;password'",
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
		cli.StringSliceFlag{
			Name:  "set",
			Usage: "set values on the command line (use --set multiple times or use commas: key1=val1,key2=val2)",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			// Build options
			buildContext    = context.Args().First()
			dockerfile      = context.String("file")
			target          = context.String("target")
			isolation       = context.String("isolation")
			platform        = context.String("platform")
			tags            = context.StringSlice("tag")
			buildArgs       = context.StringSlice("build-arg")
			secretBuildArgs = context.StringSlice("secret-build-arg")
			labels          = context.StringSlice("label")
			creds           = context.StringSlice("credential")
			pull            = context.Bool("pull")
			noCache         = context.Bool("no-cache")
			push            = context.Bool("push")
			dryRun          = context.Bool("dry-run")
			debug           = context.Bool("debug")

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

		renderOpts := &templating.BaseRenderOptions{
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
			Architecture:            runtime.GOARCH,
		}

		task, err := createBuildTask(
			ctx,
			isolation,
			pull,
			labels,
			noCache,
			dockerfile,
			tags,
			buildArgs,
			secretBuildArgs,
			target,
			platform,
			buildContext,
			renderOpts,
			debug,
			registry,
			push,
			creds)
		if err != nil {
			return err
		}

		timeout := time.Duration(task.TotalTimeout) * time.Second
		ctx, cancel := gocontext.WithTimeout(gocontext.Background(), timeout)
		defer cancel()

		builder := builder.NewBuilder(pm, debug, homevol)
		defer builder.CleanTask(gocontext.Background(), task) // Use a separate context since the other may have expired.
		return builder.RunTask(ctx, task)
	},
}

func createBuildTask(
	ctx gocontext.Context,
	isolation string,
	pull bool,
	labels []string,
	noCache bool,
	dockerfile string,
	tags []string,
	buildArgs []string,
	secretBuildArgs []string,
	target string,
	platform string,
	buildContext string,
	renderOpts *templating.BaseRenderOptions,
	debug bool,
	registry string,
	push bool,
	creds []string,
) (*graph.Task, error) {
	// Create the run command to be used in the template
	args := []string{}
	if isolation != "" {
		args = append(args, fmt.Sprintf("--isolation=%s", isolation))
	}
	if pull {
		args = append(args, "--pull")
	}
	for _, label := range labels {
		args = append(args, "--label", label)
	}
	if noCache {
		args = append(args, "--no-cache")
	}
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}
	for _, tag := range tags {
		args = append(args, "-t", tag)
	}
	for _, buildArg := range buildArgs {
		args = append(args, "--build-arg", buildArg)
	}
	for _, secretBuildArg := range secretBuildArgs {
		args = append(args, "--build-arg", secretBuildArg)
	}
	if target != "" {
		args = append(args, "--target", target)
	}
	if platform != "" {
		args = append(args, "--platform", platform)
	}
	args = append(args, buildContext)
	runCmd := strings.Join(args, " ")

	// Create the template
	template := templating.NewTemplate("build", []byte(runCmd))

	rendered, err := templating.LoadAndRenderSteps(ctx, template, renderOpts)
	if err != nil {
		return nil, err
	}

	if debug {
		log.Println("Rendered template:")
		log.Println(rendered)
	}

	// After the template has rendered, we have to parse the tags again
	// so we can properly set the build/push tags.
	rendered, prefixedTags := util.PrefixTags(rendered, registry)
	tags = prefixedTags

	buildStep := &graph.Step{
		ID:      "build",
		Build:   rendered,
		Timeout: buildTimeoutInSec,
	}

	steps := []*graph.Step{buildStep}

	if push {
		pushStep := &graph.Step{
			ID:      "push",
			Push:    tags,
			Timeout: pushTimeoutInSec,
			When:    []string{buildStep.ID},
		}

		steps = append(steps, pushStep)
	}

	var credentials []*graph.Credential
	for _, credString := range creds {
		cred, err := graph.CreateCredentialFromString(credString)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, cred)
	}

	return graph.NewTask(steps, []*graph.Secret{}, registry, credentials, taskTotalTimeoutInSec, true)
}
