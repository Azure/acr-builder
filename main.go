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

const defaultCloneDir = "/checkout"

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var composeFile string
	var gitURL string
	var gitCloneDir, gitbranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken string
	var localSource string

	// Untested code paths:
	// required unless the host is properly logged in
	// if the program is launched in docker container, use option -v /var/run/docker.sock:/var/run/docker.sock -v ~/.docker:/root/.docker
	var dockeruser, dockerpw, dockerRegistry string
	var buildArgs, buildEnvs stringSlice
	var push bool
	var buildNumber string
	flag.StringVar(&buildNumber, "build-number", "0", fmt.Sprintf("Build number, this argument would set the reserved %s build environment.", constants.BuildNumberVar))
	flag.StringVar(&gitURL, "git-url", "", "Git url to the project")
	flag.StringVar(&gitCloneDir, "git-clone-to", defaultCloneDir, "Directory to clone to. If the directory exists, we won't clone again and will just clean and pull the directory")
	flag.StringVar(&gitbranch, "git-branch", "", "The git branch to checkout. If it is not given, no checkout command would be performed.")
	flag.StringVar(&gitHeadRev, "git-head-revision", "", "Desired git HEAD revision, note that providing this parameter will cause the branch parameter to be ignored")
	flag.StringVar(&gitPATokenUser, "git-pa-token-user", "", "Git username for the personal access token.")
	flag.StringVar(&gitPAToken, "git-pa-token", "", "Git personal access token.")
	flag.StringVar(&gitXToken, "git-x-token", "", "Git OAuth x access token.")
	flag.StringVar(&localSource, "local-source", "", "Local source directory. Specifying this parameter tells the builder no source control is used and it would use the specified directory as source")
	flag.StringVar(&composeFile, "compose-file", "", "Path to the docker-compose file.")
	flag.StringVar(&dockerRegistry, "docker-registry", "", "Docker registry to push to")
	flag.StringVar(&dockeruser, "docker-user", "", "Docker username.")
	flag.StringVar(&dockerpw, "docker-password", "", "Docker password or OAuth identity token.")
	flag.Var(&buildArgs, "docker-build-arg", "Build arguments to be passed to docker build or docker-compose build")
	flag.Var(&buildEnvs, "build-env", "Custom environment variables defined for the build process")
	flag.BoolVar(&push, "push", false, "Push on success")
	flag.Parse()

	if push && dockerRegistry == "" {
		panic("Registry needs to be provided if push is needed")
	}

	if gitHeadRev != "" && gitbranch != "" {
		logrus.Infof("Both HEAD revision %s and branch %s are provided as parameter, HEAD will take precedence")
	}

	err := build.Run(buildNumber, composeFile, dockeruser, dockerpw, dockerRegistry, gitURL, gitCloneDir, gitbranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken, localSource, buildEnvs, buildArgs, push)
	ensureNoError("%s", err)
}

func ensureNoError(msg string, err error) {
	if err != nil {
		logrus.Errorf(msg, err)
		os.Exit(-1)
	}
}
