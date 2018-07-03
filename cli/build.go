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
	"github.com/Azure/acr-builder/volume"
	"github.com/google/uuid"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const buildLongDesc = `
This command can be used to build images.
`

type buildCmd struct {
	out              io.Writer
	context          string
	dockerfile       string
	target           string
	tags             []string
	buildArgs        []string
	secretBuildArgs  []string
	registry         string
	registryUserName string
	registryPassword string
	pull             bool
	noCache          bool
	push             bool
	isolation        string
	oci              bool
	dryRun           bool
}

func newBuildCmd(out io.Writer) *cobra.Command {
	r := &buildCmd{
		out: out,
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

	f.StringVarP(&r.registry, "registry", "r", "", "the name of the registry")
	f.StringVarP(&r.registryUserName, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&r.registryPassword, "password", "p", "", "the password to use when logging into the registry")

	f.StringVar(&r.isolation, "isolation", "default", "the isolation to use")
	f.StringVar(&r.target, "target", "", "specify a stage to build")
	f.BoolVar(&r.pull, "pull", false, "attempt to pull a newer version of the base images")
	f.BoolVar(&r.noCache, "no-cache", false, "true to ignore all cached layers when building the image")
	f.BoolVar(&r.push, "push", false, "push on success")
	f.BoolVar(&r.oci, "oci", false, "use the OCI builder")
	f.BoolVar(&r.dryRun, "dry-run", false, "evaluates the build but doesn't execute it")

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

	normalizedDockerImages := builder.GetNormalizedDockerImageNames(b.tags)

	cmder := cmder.NewCmder(b.dryRun)

	defaultStep := &graph.Step{
		UseLocalContext: true,
		Run:             b.createRunCmd(),
	}

	steps := []*graph.Step{defaultStep}

	push := []string{}
	if b.push {
		for _, img := range normalizedDockerImages {
			push = append(push, img)
		}
	}

	// TODO: create secrets
	secrets := []*graph.Secret{}

	p := graph.NewPipeline(steps, push, secrets)
	dag, err := graph.NewDagFromPipeline(p)
	if err != nil {
		return err
	}

	timeout := time.Duration(p.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
	fmt.Printf("Setting up the home volume: %s...\n", homeVolName)
	v := volume.NewVolume(homeVolName, cmder)
	if err := v.Create(ctx); err != nil {
		return fmt.Errorf("Err creating docker vol: %v", err)
	}
	defer func() {
		if err := v.Delete(ctx); err != nil {
			fmt.Printf("Failed to clean up docker vol: %s. Err: %v\n", homeVolName, err)
		}
	}()

	bo := &builder.BuildOptions{
		RegistryName:     b.registry,
		RegistryUsername: b.registryUserName,
		RegistryPassword: b.registryPassword,
		Pull:             b.pull,
		Push:             b.push,
		NoCache:          b.noCache,
	}

	builder := builder.NewBuilder(cmder, debug, homeVolName, bo)
	defer builder.CleanAllBuildSteps(context.Background(), dag)

	return builder.RunAllBuildSteps(ctx, dag, p.Push)
}

func (b *buildCmd) validateCmdArgs() error {
	if err := validateIsolation(b.isolation); err != nil {
		return err
	}

	if err := validateRegistryCreds(b.registryUserName, b.registryPassword); err != nil {
		return err
	}

	if err := validatePush(b.push, b.registry, b.registryUserName, b.registryPassword); err != nil {
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
		args = append(args)
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

	args = append(args, b.context)
	return strings.Join(args, " ")
}
