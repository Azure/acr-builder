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
}

func newBuildCmd(out io.Writer) *cobra.Command {
	r := &buildCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Run a build",
		Long:  buildLongDesc,
		RunE:  r.run,
	}

	f := cmd.Flags()

	// TODO: support for build envs

	// Build parameters
	// TODO: support positional argument for context
	f.StringVarP(&r.context, "context", "c", ".", "context used to build")
	f.StringVarP(&r.dockerfile, "dockerfile", "f", "Dockerfile", "name of the Dockerfile")
	f.StringArrayVarP(&r.tags, "tag", "t", []string{}, "name and optionally a tag in the 'name:tag' format")
	f.StringArrayVarP(&r.buildArgs, "arg", "a", []string{}, "set build time arguments")
	f.StringArrayVarP(&r.secretBuildArgs, "secret-arg", "s", []string{}, "set secret build arguments")

	f.StringVarP(&r.registry, "registry", "r", "", "the name of the registry")
	f.StringVarP(&r.registryUserName, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&r.registryPassword, "password", "p", "", "the password to use when logging into the registry")

	f.StringVar(&r.isolation, "isolation", "default", "the isolation to use")
	f.BoolVar(&r.pull, "pull", false, "attempt to pull a newer version of the base images")
	f.BoolVar(&r.noCache, "no-cache", false, "true to ignore all cached layers when building the image")
	f.BoolVar(&r.push, "push", false, "push on success")
	f.BoolVar(&r.oci, "oci", false, "use the OCI builder")

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return cmd
}

func (b *buildCmd) run(cmd *cobra.Command, args []string) error {
	if err := b.validateCmdArgs(); err != nil {
		return err
	}

	normalizedDockerImages := getNormalizedDockerImageNames(b.tags)

	cmder := cmder.NewCmder(false)

	defaultStep := &graph.Step{
		ID:  "Build",
		Run: b.createRunCmd(),
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

	bo := &builder.BuildOptions{
		RegistryName:     b.registry,
		RegistryUsername: b.registryUserName,
		RegistryPassword: b.registryPassword,
		Pull:             b.pull,
		Push:             b.push,
		NoCache:          b.noCache,
	}

	builder := builder.NewBuilder(cmder, debug, "", false, bo)
	defer builder.CleanAllBuildSteps(context.Background())

	return builder.RunAllBuildSteps(ctx, dag, p.Push)
}

func (b *buildCmd) validateCmdArgs() error {
	if err := validateIsolation(b.isolation); err != nil {
		return err
	}

	if err := validateRegistryCreds(b.registryUserName, b.registryPassword); err != nil {
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

	args = append(args, b.context)
	return strings.Join(args, " ")
}
