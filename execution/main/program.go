package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/acr-builder/execution/builder"
	"github.com/Azure/acr-builder/execution/constants"
	"github.com/sirupsen/logrus"
)

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	// required
	var composeFile string

	// required when using git as source
	var gitURL string
	// optional when using git as source
	var gitCloneDir, gitbranch, gitOAuthUser, gitOAuthToken string

	// Untested code paths:
	// required unless the host is properly logged in
	// if the program is launched in docker container, use option -v /var/run/docker.sock:/var/run/docker.sock -v ~/.docker:/root/.docker
	var dockeruser, dockerpw string
	// optional
	var registry string
	var buildArgs, buildEnvs stringSlice
	var nopublish bool
	var buildNumber string
	flag.StringVar(&buildNumber, "build-number", "0", fmt.Sprintf("Build number, this argument would set the reserved %s build environment.", constants.BuildNumberVar))
	flag.StringVar(&gitURL, "git-url", "", "Git url to the project")
	flag.StringVar(&gitCloneDir, "git-clone-to", constants.DefaultCloneDir, "Directory to clone to. If the directory exists, we won't clone again and will just clean and pull the directory")
	flag.StringVar(&gitbranch, "git-branch", "", "The git branch to checkout. If it is not given, no checkout command would be performed.")
	flag.StringVar(&gitOAuthUser, "git-oath-user", "", "Git username.")
	flag.StringVar(&gitOAuthToken, "git-oath-token", "", "Git personal access token.")
	flag.StringVar(&composeFile, "compose-file", "", "Path to the docker-compose file.")
	flag.StringVar(&registry, "docker-registry", "", "Docker registry to publish to")
	flag.StringVar(&dockeruser, "docker-user", "", "Docker username.")
	flag.StringVar(&dockerpw, "docker-password", "", "Docker password or OAuth identity token.")
	flag.Var(&buildArgs, "docker-build-arg", "Build arguments to be passed to docker build or docker-compose build")
	flag.Var(&buildEnvs, "build-env", "Custom environment variables defined for the build process")
	flag.BoolVar(&nopublish, "no-publish", false, "Do not publish on success")
	flag.Parse()

	if registry == "" {
		panic("Registry needs to be provided")
	}
	err := build.Run(buildNumber, composeFile, gitURL, dockeruser, dockerpw, registry, gitCloneDir, gitbranch, gitOAuthUser, gitOAuthToken, buildEnvs, buildArgs, nopublish)
	ensureNoError("%s", err)
}

func ensureNoError(msg string, err error) {
	if err != nil {
		logrus.Errorf(msg, err)
		os.Exit(-1)
	}
}
