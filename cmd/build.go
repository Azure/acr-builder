// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/acr-builder/builder"
	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/taskmanager"
	"github.com/Azure/acr-builder/templating"
	"github.com/Azure/acr-builder/util"
	"github.com/Azure/acr-builder/volume"
	"github.com/google/uuid"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const buildLongDesc = `
This command can be used to build images.
`

type buildCmd struct {
	out             io.Writer
	context         string
	dockerfile      string
	target          string
	registryUser    string
	registryPw      string
	isolation       string
	network         string
	platform        string
	tags            []string
	buildArgs       []string
	secretBuildArgs []string
	labels          []string
	pull            bool
	noCache         bool
	push            bool
	oci             bool
	dryRun          bool

	opts *templating.BaseRenderOptions
}

func newBuildCmd(out io.Writer) *cobra.Command {
	r := &buildCmd{
		out:  out,
		opts: &templating.BaseRenderOptions{},
	}

	cmd := &cobra.Command{
		Use:   "build [OPTIONS] PATH | URL",
		Short: "Run a build",
		Long:  buildLongDesc,
		RunE:  r.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("acb build requires exactly 1 argument. See acb build --help")
			}

			return nil
		},
	}

	f := cmd.Flags()

	// Build parameters
	f.StringVarP(&r.dockerfile, "file", "f", "Dockerfile", "name of the Dockerfile")
	f.StringArrayVarP(&r.tags, "tag", "t", []string{}, "name and optionally a tag in the 'name:tag' format")
	f.StringArrayVar(&r.buildArgs, "build-arg", []string{}, "set build time arguments")
	f.StringArrayVar(&r.secretBuildArgs, "secret-build-arg", []string{}, "set secret build arguments")
	f.StringArrayVar(&r.labels, "label", []string{}, "set metadata for an image")

	f.StringVarP(&r.registryUser, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&r.registryPw, "password", "p", "", "the password to use when logging into the registry")

	f.StringVar(&r.isolation, "isolation", "", "the isolation to use")
	f.StringVar(&r.network, "network", "", "set the networking mode during build")
	f.StringVar(&r.target, "target", "", "specify a stage to build")
	f.StringVar(&r.platform, "platform", "", "sets the platform if the server is capable of multiple platforms")
	f.BoolVar(&r.pull, "pull", false, "attempt to pull a newer version of the base images")
	f.BoolVar(&r.noCache, "no-cache", false, "true to ignore all cached layers when building the image")
	f.BoolVar(&r.push, "push", false, "push on success")
	f.BoolVar(&r.oci, "oci", false, "use the OCI builder")
	f.BoolVar(&r.dryRun, "dry-run", false, "evaluates the build but doesn't execute it")

	AddBaseRenderingOptions(f, r.opts, cmd, false)

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return cmd
}

func (b *buildCmd) run(cmd *cobra.Command, args []string) error {
	if err := b.validateCmdArgs(); err != nil {
		return err
	}

	b.context = args[0]

	template := &templating.Template{
		Name: "build",
		Data: []byte(b.createRunCmd()),
	}

	rendered, err := templating.LoadAndRenderSteps(template, b.opts)
	if err != nil {
		return err
	}

	if debug {
		fmt.Println("Rendered template:")
		fmt.Println(rendered)
	}

	// After the template has rendered, we have to parse the tags again
	// so we can properly set the push targets.
	b.tags = util.ParseTags(rendered)

	taskManager := taskmanager.NewTaskManager(b.dryRun)
	defaultStep := &graph.Step{
		UseLocalContext: true,
		Run:             rendered,
	}

	steps := []*graph.Step{defaultStep}

	push := []string{}
	if b.push {
		for _, img := range b.tags {
			push = append(push, img)
		}
	}

	// TODO: create secrets
	secrets := []*graph.Secret{}

	pipeline, err := graph.NewPipeline(steps, push, secrets, b.opts.Registry, b.registryUser, b.registryPw)
	if err != nil {
		return err
	}

	timeout := time.Duration(pipeline.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
	if !b.dryRun {
		fmt.Printf("Setting up the home volume: %s...\n", homeVolName)
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

func (b *buildCmd) validateCmdArgs() error {
	if err := validateIsolation(b.isolation); err != nil {
		return err
	}

	if err := validateRegistryCreds(b.registryUser, b.registryPw); err != nil {
		return err
	}

	if err := validatePush(b.push, b.opts.Registry, b.registryUser, b.registryPw); err != nil {
		return err
	}

	// TODO: OCI build support
	if b.oci {
		return errors.New("OCI builder isn't implemented yet")
	}

	return nil
}

func (b *buildCmd) createRunCmd() string {
	args := []string{"build"}
	if b.isolation != "" {
		args = append(args, fmt.Sprintf("--isolation=%s", b.isolation))
	}

	if b.pull {
		args = append(args, "--pull")
	}

	if b.network != "" {
		args = append(args, fmt.Sprintf("--network=%s", b.network))
	}

	for _, label := range b.labels {
		args = append(args, "--label", label)
	}

	if b.noCache {
		args = append(args, "--no-cache")
	}

	if b.dockerfile != "" {
		args = append(args, "-f", b.dockerfile)
	}

	for _, imgName := range b.tags {
		args = append(args, "-t", imgName)
	}

	for _, buildArg := range b.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	for _, buildSecretArg := range b.secretBuildArgs {
		args = append(args, "--build-arg", buildSecretArg)
	}

	if b.target != "" {
		args = append(args, "--target", b.target)
	}

	if b.platform != "" {
		args = append(args, "--platform", b.platform)
	}

	args = append(args, b.context)
	return strings.Join(args, " ")
}
