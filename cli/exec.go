package cli

import (
	"context"
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
	out       io.Writer
	dryRun    bool
	stepsFile string

	registry         string
	registryUserName string
	registryPassword string

	// Build-time parameters for rendering
	templatePath     string
	templateValues   []string
	buildID          string
	buildCommit      string
	buildTag         string
	buildRepository  string
	buildBranch      string
	buildTriggeredBy string
}

func newExecCmd(out io.Writer) *cobra.Command {
	e := &execCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute a pipeline",
		Long:  execLongDesc,
		RunE:  e.run,
	}

	f := cmd.Flags()

	f.StringVar(&e.stepsFile, "steps", "", "the steps file to use when building")

	f.StringVarP(&e.registry, "registry", "r", "", "the name of the registry")
	f.StringVarP(&e.registryUserName, "username", "u", "", "the username to use when logging into the registry")
	f.StringVarP(&e.registryPassword, "password", "p", "", "the password to use when logging into the registry")

	// TODO: better names and shorthand
	// Rendering parameters
	f.StringVar(&e.buildID, "id", "", "the build ID")
	f.StringVarP(&e.buildCommit, "commit", "c", "", "the commit SHA")
	f.StringVarP(&e.buildTag, "tag", "t", "", "the build tag")
	f.StringVar(&e.buildRepository, "repository", "", "the build repository")
	f.StringVarP(&e.buildBranch, "branch", "b", "", "the build branch")
	f.StringVar(&e.buildTriggeredBy, "triggered-by", "", "what the build was triggered by")
	f.StringVar(&e.templatePath, "template-path", "", "the path to the job to render")
	f.StringArrayVar(&e.templateValues, "set", []string{}, "set values on the command line (use `--set` multiple times or use commas: key1=val1,key2=val2)")
	f.BoolVar(&e.dryRun, "dryrun", false, "evaluates the pipeline but doesn't execute it")

	return cmd
}

func (e *execCmd) run(cmd *cobra.Command, args []string) error {
	cmder := cmder.NewCmder(e.dryRun)

	ctx := context.Background()

	if !e.dryRun {
		clientVersion, serverVersion, err := cmder.GetDockerVersions(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("Using Docker client version: %s, server version: %s\n", clientVersion, serverVersion)
	}

	j, err := templating.LoadJob(e.templatePath)
	if err != nil {
		return fmt.Errorf("Failed to load job at path %s. Err: %v", e.templatePath, err)
	}
	engine := templating.New()

	bo := templating.BaseRenderOptions{
		ID:          e.buildID,
		Commit:      e.buildCommit,
		Tag:         e.buildTag,
		Repository:  e.buildRepository,
		Branch:      e.buildBranch,
		TriggeredBy: e.buildTriggeredBy,
	}

	rawVals, err := combineVals(e.templateValues)
	if err != nil {
		return err
	}

	config := &templating.Config{RawValue: rawVals, Values: map[string]*templating.Value{}}
	vals, err := templating.OverrideValuesWithBuildInfo(j, config, bo)
	if err != nil {
		return fmt.Errorf("Failed to override values: %v", err)
	}

	expectedTmplName := fmt.Sprintf("templates/%s", e.stepsFile)

	keep := map[string]bool{expectedTmplName: true}

	rendered, err := engine.Render(j, vals, keep)
	if err != nil {
		return fmt.Errorf("Error while rendering templates: %v", err)
	}

	p, err := graph.UnmarshalPipelineFromString(rendered[expectedTmplName])
	if err != nil {
		return err
	}

	dag, err := graph.NewDagFromPipeline(p)
	if err != nil {
		return err
	}

	timeout := time.Duration(p.TotalTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	homeVolName := fmt.Sprintf("%s%s", volume.VolumePrefix, uuid.New())
	if !e.dryRun {
		v := volume.NewVolume(homeVolName, cmder)
		if err := v.Create(ctx); err != nil {
			return fmt.Errorf("Err creating docker vol: %v", err)
		}
		defer func() {
			if err := v.Delete(ctx); err != nil {
				fmt.Printf("Failed to clean up docker vol: %s. Err: %v\n", homeVolName, err)
			}
		}()
	}

	buildOptions := &builder.BuildOptions{
		RegistryName:     e.registry,
		RegistryUsername: e.registryUserName,
		RegistryPassword: e.registryPassword,
		Push:             len(p.Push) > 0,
	}

	builder := builder.NewBuilder(cmder, debug, homeVolName, e.dryRun, buildOptions)
	defer builder.CleanAllBuildSteps(context.Background())

	return builder.RunAllBuildSteps(ctx, dag)
}

func combineVals(values []string) (string, error) {
	ret := templating.Values{}

	// TODO: support passing in multiple value files?

	// User specified a value via --set
	for _, v := range values {
		s := strings.Split(v, "=")
		if len(s) != 2 {
			return "", fmt.Errorf("failed to parse --set data: %s", v)
		}
		ret[s[0]] = s[1]
	}

	return ret.ToTOMLString()
}
