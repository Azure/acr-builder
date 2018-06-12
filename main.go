package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/driver"
	"github.com/Azure/acr-builder/pkg/shell"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/sirupsen/logrus"
)

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	// Don't allow empty values
	if value != "" {
		*i = append(*i, value)
	}
	return nil
}

var (
	help        = flag.Bool("help", false, "Prints the help message")
	versionFlag = flag.Bool("version", false, "Prints the version of the builder")
)

func main() {
	var dockerfile, dockerContextString string
	var dockerImages stringSlice
	// Untested code paths:
	// required unless the host is properly logged in
	// if the program is launched in docker container, use option -v /var/run/docker.sock:/var/run/docker.sock -v ~/.docker:/root/.docker
	var dockerUser, dockerPW, dockerRegistry string
	var buildArgs, buildSecretArgs, buildEnvs stringSlice
	var hypervIsolation, pull, noCache, push, debug bool
	flag.StringVar(&dockerContextString, constants.ArgNameDockerContextString, "", "Working directory for the builder.")
	flag.StringVar(&dockerfile, constants.ArgNameDockerfile, "", "Dockerfile to build. If choosing to build a dockerfile")
	flag.Var(&dockerImages, constants.ArgNameDockerImage, "The image names to build to. This option is only available when building with dockerfile")
	flag.Var(&buildArgs, constants.ArgNameDockerBuildArg, "Build arguments to be passed to docker build build")
	flag.Var(&buildSecretArgs, constants.ArgNameDockerSecretBuildArg, "Build arguments to be passed to docker build build. The argument value contains a secret which will be hidden from the log.")
	flag.StringVar(&dockerRegistry, constants.ArgNameDockerRegistry, "", "Docker registry to push to")
	flag.StringVar(&dockerUser, constants.ArgNameDockerUser, "", "Docker username.")
	flag.StringVar(&dockerPW, constants.ArgNameDockerPW, "", "Docker password or OAuth identity token.")
	flag.Var(&buildEnvs, constants.ArgNameBuildEnv, "Custom environment variables defined for the build process")
	flag.BoolVar(&pull, constants.ArgNamePull, false, "Attempt to pull a newer version of the base images")
	flag.BoolVar(&hypervIsolation, constants.ArgNameHypervIsolation, false, "Build using Hyper-V hypervisor partition based isolation")
	flag.BoolVar(&noCache, constants.ArgNameNoCache, false, "Not using any cached layer when building the image")
	flag.BoolVar(&push, constants.ArgNamePush, false, "Push on success")
	flag.BoolVar(&debug, constants.ArgNameDebug, false, "Enable verbose output for debugging")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}

	if *versionFlag {
		fmt.Printf(`%s:
				go version  : %s
				go compiler : %s
				platform    : %s/%s
			   `, os.Args[0], runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
		return
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	runner := shell.NewRunner()
	defer runner.GetFileSystem().Cleanup()
	builder := driver.NewBuilder(runner)

	normalizedDockerImages := getNormalizedDockerImageNames(dockerImages)

	dependencies, err := builder.Run(
		dockerfile, normalizedDockerImages,
		dockerUser, dockerPW, dockerRegistry,
		dockerContextString,
		buildEnvs, buildArgs, buildSecretArgs, hypervIsolation, pull, noCache, push)

	if err != nil {
		logrus.Error(err)
		if len(os.Args) < 2 {
			flag.CommandLine.Usage()
		}
		os.Exit(constants.GeneralErrorExitCode)
	}

	output, err := json.Marshal(dependencies)
	if err != nil {
		logrus.Errorf("Failed to serialize dependencies %s", err)
		os.Exit(constants.GeneralErrorExitCode)
	}

	fmt.Printf("\nACR Builder discovered the following dependencies:\n%s\n", string(output))
	fmt.Println("\nBuild complete")
}

// getNormalizedDockerImageNames normalizes the list of docker images
// and removes any duplicates.
func getNormalizedDockerImageNames(dockerImages []string) []string {
	dict := map[string]bool{}
	normalizedDockerImages := []string{}
	for _, d := range dockerImages {
		d := build.NormalizeImageTag(d)
		if dict[d] {
			continue
		}

		dict[d] = true
		normalizedDockerImages = append(normalizedDockerImages, d)
	}

	return normalizedDockerImages
}
