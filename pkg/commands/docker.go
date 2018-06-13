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
	dockerbuild "github.com/docker/cli/cli/command/image/build"
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
func NewDockerBuild(dockerfile string,
	buildArgs, buildSecretArgs []string, registry string, imageNames []string, isolation string, pull, noCache bool) build.Target {
	var pushTo []string
	// If imageName is empty, skip push.
	// If registry is empty, it means push to DockerHub.
	for _, imageName := range imageNames {
		pushTarget := imageName

		// If the registry's specified and the image name is already prefixed with
		// the registry's name, don't prefix the registry name again.
		if registry != "" && !strings.HasPrefix(imageName, registry) {
			pushTarget = fmt.Sprintf("%s%s", registry, imageName)
		}

		pushTo = append(pushTo, pushTarget)
	}

	return &dockerBuildTask{
		dockerfile:      dockerfile,
		buildArgs:       buildArgs,
		buildSecretArgs: buildSecretArgs,
		pushTo:          pushTo,
		isolation:       isolation,
		pull:            pull,
		noCache:         noCache,
	}
}

type dockerBuildTask struct {
	dockerfile      string
	buildArgs       []string
	buildSecretArgs []string
	pushTo          []string
	isolation       string
	pull            bool
	noCache         bool
}

func (t *dockerBuildTask) Ensure(runner build.Runner) error {
	if t.dockerfile == constants.FromStdin {
		t.dockerfile = dockerbuild.DefaultDockerfileName
		return runner.GetFileSystem().WriteFile(t.dockerfile, runner.GetStdin())
	}
	return nil
}

func (t *dockerBuildTask) ScanForDependencies(runner build.Runner) ([]build.ImageDependencies, error) {
	env := runner.GetContext()
	var dockerfile string
	var dep []build.ImageDependencies
	if t.dockerfile == "" {
		dockerfile = dockerbuild.DefaultDockerfileName
	} else {
		dockerfile = env.Expand(t.dockerfile)
	}
	runtime, buildtime, err := grok.ResolveDockerfileDependencies(t.buildArgs, dockerfile)
	if err != nil {
		return nil, err
	}

	// Even though there's nothing to push to, we always invoke NewImageDependencies
	// TODO: refactor this in the future to take in the full list as opposed to individual
	// images.
	if len(t.pushTo) <= 0 {
		currDep, err := build.NewImageDependencies(env, "", runtime, buildtime)
		if err != nil {
			return nil, err
		}
		dep = append(dep, *currDep)
	}

	for _, imageName := range t.pushTo {
		currDep, err := build.NewImageDependencies(env, imageName, runtime, buildtime)
		if err != nil {
			return nil, err
		}
		dep = append(dep, *currDep)
	}

	return dep, err
}

func (t *dockerBuildTask) Build(runner build.Runner) error {
	args := []string{"build"}

	if t.isolation != "" {
		isolationString := fmt.Sprintf("--isolation=%s", t.isolation)
		args = append(args, isolationString)
	}

	if t.pull {
		args = append(args, "--pull")
	}

	if t.noCache {
		args = append(args, "--no-cache")
	}

	if t.dockerfile != "" {
		args = append(args, "-f", t.dockerfile)
	}

	for _, imageName := range t.pushTo {
		args = append(args, "-t", imageName)
	}

	for _, buildArg := range t.buildArgs {
		args = append(args, "--build-arg", buildArg)
	}

	for _, buildSecretArg := range t.buildSecretArgs {
		args = append(args, "--build-arg", buildSecretArg)
	}

	args = append(args, ".")

	return runner.ExecuteCmdWithObfuscation(KeyValueArgumentObfuscator(t.buildSecretArgs), "docker", args)
}

func (t *dockerBuildTask) Export() []build.EnvVar {
	return []build.EnvVar{
		{
			Name:  constants.ExportsDockerfilePath,
			Value: t.dockerfile,
		},
	}
}

func (t *dockerBuildTask) Push(runner build.Runner) error {
	if len(t.pushTo) <= 0 {
		return fmt.Errorf("No push target is defined")
	}

	for _, imageName := range t.pushTo {
		err := runner.ExecuteCmd("docker", []string{"push", imageName}, nil)
		if err != nil {
			return err
		}
	}

	return nil
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

		// refString will always have the tag specified at this point.
		// For "scratch", we have to compare it against "scratch:latest" even though
		// scratch:latest isn't valid in a FROM clause.
		if refString == constants.NoBaseImageSpecifierLatest {
			return nil
		}

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
	// If the reference has "library/" prefixed, we have to remove it - otherwise
	// we'll fail to query the digest, since image names aren't prefixed with "library/"
	if strings.HasPrefix(prefix, "library/") && reference.Registry == build.DockerHubRegistry {
		prefix = prefix[8:]
	} else if len(reference.Registry) > 0 && reference.Registry != build.DockerHubRegistry {
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
