package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/grok"
)

const (
	dockerLoginTimeout = time.Second * 30
)

// NewDockerUsernamePassword creates a authentication object with username and password
func NewDockerUsernamePassword(registry string, username string, password string) (build.DockerCredential, error) {
	if (username == "") != (password == "") {
		return nil, fmt.Errorf("Please provide both --%s and --%s or neither", constants.ArgNameDockerUser, constants.ArgNameDockerPW)
	}
	return &dockerUsernamePassword{
		registry: registry,
		username: username,
		password: password,
	}, nil
}

type dockerUsernamePassword struct {
	registry string
	username string
	password string
}

func (u *dockerUsernamePassword) Authenticate(runner build.Runner) error {
	c := make(chan error, 1)

	go func() {
		c <- runner.ExecuteCmd("docker", []string{"login", "-u", u.username, "--password-stdin", u.registry}, strings.NewReader(u.password+"\n"))
	}()

	select {
	case err := <-c:
		return err
	case <-time.After(dockerLoginTimeout):
		err := fmt.Errorf("docker login timed out")
		return err
	}
}

// NewDockerBuild creates a build target with specified docker file and build parameters
func NewDockerBuild(dockerfile, contextDir string,
	buildArgs, buildSecretArgs []string, registry, imageName string, pull, noCache bool) build.Target {
	return &dockerBuildTask{
		dockerfile:      dockerfile,
		contextDir:      contextDir,
		buildArgs:       buildArgs,
		buildSecretArgs: buildSecretArgs,
		pushTo:          fmt.Sprintf("%s%s", registry, imageName),
		pull:            pull,
		noCache:         noCache,
	}
}

type dockerBuildTask struct {
	dockerfile      string
	contextDir      string
	buildArgs       []string
	buildSecretArgs []string
	pushTo          string
	pull            bool
	noCache         bool
}

func (t *dockerBuildTask) ScanForDependencies(runner build.Runner) ([]build.ImageDependencies, error) {
	env := runner.GetContext()
	var dockerfile string
	if t.dockerfile == "" {
		dockerfile = constants.DefaultDockerfile
	} else {
		dockerfile = env.Expand(t.dockerfile)
	}
	runtime, buildtime, err := grok.ResolveDockerfileDependencies(t.buildArgs, dockerfile)
	if err != nil {
		return nil, err
	}
	dep, err := build.NewImageDependencies(env, t.pushTo, runtime, buildtime)
	if err != nil {
		return nil, err
	}
	return []build.ImageDependencies{*dep}, err
}

func (t *dockerBuildTask) Build(runner build.Runner) error {
	args := []string{"build"}

	if t.pull {
		args = append(args, "--pull")
	}

	if t.noCache {
		args = append(args, "--no-cache")
	}

	if t.dockerfile != "" {
		args = append(args, "-f", t.dockerfile)
	}

	if t.pushTo != "" {
		args = append(args, "-t", t.pushTo)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	for _, buildSecretArg := range t.buildSecretArgs {
		args = append(args, "--build-arg", buildSecretArg)
	}

	if t.contextDir != "" {
		args = append(args, t.contextDir)
	} else {
		args = append(args, ".")
	}

	return runner.ExecuteCmdWithObfuscation(KeyValueArgumentObfuscator(t.buildSecretArgs), "docker", args)
}

func (t *dockerBuildTask) Export() []build.EnvVar {
	return []build.EnvVar{
		{
			Name:  constants.ExportsDockerfilePath,
			Value: t.dockerfile,
		},
		{
			Name:  constants.ExportsDockerBuildContext,
			Value: t.contextDir,
		},
		{
			Name:  constants.ExportsDockerPushImage,
			Value: t.pushTo,
		},
	}
}

func (t *dockerBuildTask) Push(runner build.Runner) error {
	if t.pushTo == "" {
		return fmt.Errorf("No push target is defined")
	}
	return runner.ExecuteCmd("docker", []string{"push", t.pushTo}, nil)
}

// PopulateDigests populates digests on dependencies
func PopulateDigests(runner build.Runner, dependencies []build.ImageDependencies) error {
	for _, entry := range dependencies {
		if err := queryDigest(runner, entry.Image); err != nil {
			return err
		}
		if err := queryDigest(runner, entry.Runtime); err != nil {
			return err
		}
		for _, buildtime := range entry.Buildtime {
			if err := queryDigest(runner, buildtime); err != nil {
				return err
			}
		}
	}
	return nil
}

func queryDigest(runner build.Runner, reference *build.ImageReference) error {
	if reference != nil {
		refString := reference.String()
		output, err := runner.QueryCmd("docker", []string{
			"inspect", "--format", "\"{{json .RepoDigests}}\"", refString,
		})
		if err != nil {
			return err
		}

		trimCharPredicate := func(c rune) bool {
			return '\n' == c || '\r' == c || '"' == c || '\t' == c
		}

		reference.Digest = getRepoDigest(strings.TrimFunc(output, trimCharPredicate), reference)
	}

	return nil
}

func getRepoDigest(jsonContent string, reference *build.ImageReference) string {
	// Input: ["docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], , docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: ["test.azurecr.io/docker@sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce"], test.azurecr.io, docker
	// Output: sha256:b90307d28c6a6ab3d1d873d03a26c53c282bb94d5b5fb62cc7c027c384fe50ce

	// Input: Invalid
	// Output: <empty>

	prefix := reference.Repository + "@"
	if len(reference.Registry) > 0 && reference.Registry != build.DockerHubRegistry {
		prefix = reference.Registry + "/" + prefix
	}
	var digestList []string
	if err := json.Unmarshal([]byte(jsonContent), &digestList); err != nil {
		logrus.Warnf("Error deserialize %s to json, error: %s", jsonContent, err)
	}

	for _, digest := range digestList {
		if strings.HasPrefix(digest, prefix) {
			return digest[len(prefix):]
		}
	}

	logrus.Warnf("Unable to find digest for %s in %s", prefix, jsonContent)
	return ""
}
