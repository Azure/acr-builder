package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/acr-builder/pkg/shell"

	"github.com/Azure/acr-builder/pkg/build"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/sirupsen/logrus"
)

const defaultCloneDir = "$HOME/acr-builder/src"

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var composeFile, composeProjectDir string
	var dockerfile, dockerImage, dockerContextDir string
	var gitURL, gitCloneDir, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken string
	var localSource string

	// Untested code paths:
	// required unless the host is properly logged in
	// if the program is launched in docker container, use option -v /var/run/docker.sock:/var/run/docker.sock -v ~/.docker:/root/.docker
	var dockerUser, dockerPW, dockerRegistry string
	var buildArgs, buildEnvs stringSlice
	var push, debug bool
	var buildNumber string
	flag.StringVar(&buildNumber, constants.ArgNameBuildNumber, "0", fmt.Sprintf("Build number, this argument would set the reserved %s build environment.", constants.ExportsBuildNumber))
	flag.StringVar(&gitURL, constants.ArgNameGitURL, "", "Git url to the project")
	flag.StringVar(&gitCloneDir, constants.ArgNameGitCloneTo, defaultCloneDir, "Directory to clone to. If the directory exists, we won't clone again and will just clean and pull the directory")
	flag.StringVar(&gitBranch, constants.ArgNameGitBranch, "", "The git branch to checkout. If it is not given, no checkout command would be performed.")
	flag.StringVar(&gitHeadRev, constants.ArgNameGitHeadRev, "", "Desired git HEAD revision, note that providing this parameter will cause the branch parameter to be ignored")
	flag.StringVar(&gitPATokenUser, constants.ArgNameGitPATokenUser, "", "Git username for the personal access token.")
	flag.StringVar(&gitPAToken, constants.ArgNameGitPAToken, "", "Git personal access token.")
	flag.StringVar(&gitXToken, constants.ArgNameGitXToken, "", "Git OAuth x access token.")
	flag.StringVar(&localSource, constants.ArgNameLocalSource, "", "Local source directory. Specifying this parameter tells the builder no source control is used and it would use the specified directory as source")
	flag.StringVar(&composeFile, constants.ArgNameDockerComposeFile, "", "Path to the docker-compose file.")
	flag.StringVar(&composeProjectDir, constants.ArgNameDockerComposeProjectDir, "", "The --project-directory parameter for docker-compose. The default is where the compose file is")
	flag.StringVar(&dockerfile, constants.ArgNameDockerfile, "", "Dockerfile to build. If choosing to build a dockerfile")
	flag.StringVar(&dockerImage, constants.ArgNameDockerImage, "", "The image name to build to. This option is only available when building with dockerfile")
	flag.StringVar(&dockerContextDir, constants.ArgNameDockerContextDir, "", "Context directory for docker build. This option is only available when building with dockerfile.")
	flag.Var(&buildArgs, constants.ArgNameDockerBuildArg, "Build arguments to be passed to docker build or docker-compose build")
	flag.StringVar(&dockerRegistry, constants.ArgNameDockerRegistry, "", "Docker registry to push to")
	flag.StringVar(&dockerUser, constants.ArgNameDockerUser, "", "Docker username.")
	flag.StringVar(&dockerPW, constants.ArgNameDockerPW, "", "Docker password or OAuth identity token.")
	flag.Var(&buildEnvs, constants.ArgNameBuildEnv, "Custom environment variables defined for the build process")
	flag.BoolVar(&push, constants.ArgNamePush, false, "Push on success")
	flag.BoolVar(&debug, constants.ArgNameDebug, false, "Enable verbose output for debugging")
	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	builder := build.NewBuilder(shell.NewRunner())
	dep, err := builder.Run(buildNumber, composeFile, composeProjectDir,
		dockerfile, dockerImage, dockerContextDir,
		dockerUser, dockerPW, dockerRegistry,
		gitURL, gitCloneDir, gitBranch, gitHeadRev, gitPATokenUser, gitPAToken, gitXToken,
		localSource, buildEnvs, buildArgs, push)

	if err != nil {
		logrus.Error(err)
		if len(os.Args) < 2 {
			flag.CommandLine.Usage()
		}
		os.Exit(-1)
	}

	output, err := json.Marshal(dep)
	if err != nil {
		logrus.Errorf("Failed to serialize dependencies %s", err)
		os.Exit(-1)
	}
	fmt.Printf("\nACR Builder discovered the following dependencies:\n%s\n", string(output))
}
