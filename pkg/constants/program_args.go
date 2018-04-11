package constants

const (
	// ArgNameBuildNumber is the parameter name for build number
	ArgNameBuildNumber = "build-number"

	//ArgNameWorkingDir is the parameter name for the git clone target directory
	ArgNameWorkingDir = "working-dir"

	//ArgNameDockerContextString is the parameter name docker build context, it could be local path, git, or archive url
	ArgNameDockerContextString = "context"

	// ArgNameGitURL is the parameter name for git URL
	ArgNameGitURL = "git-url"

	// ArgNameGitBranch is the parameter name for git branch
	ArgNameGitBranch = "git-branch"

	// ArgNameGitHeadRev is the parameter name for head revision
	ArgNameGitHeadRev = "git-head-revision"

	// ArgNameGitPATokenUser is the parameter name for username for git username
	ArgNameGitPATokenUser = "git-username"

	// ArgNameGitPAToken is the parameter name for git password
	ArgNameGitPAToken = "git-password"

	// ArgNameGitXToken is the parameter name for github x-oath-basic token
	ArgNameGitXToken = "git-x-token"

	// ArgNameWebArchive is the parameter name for web archives
	ArgNameWebArchive = "archive"

	// ArgNameDockerfile is the parameter name for dockerfile
	ArgNameDockerfile = "docker-file"

	// ArgNameDockerImage is the parameter name for docker image (registry url must be excluded from the image name parameter)
	ArgNameDockerImage = "t"

	// ArgNameDockerContextDir is the parameter name for docker context directory
	ArgNameDockerContextDir = "docker-context-dir"

	// ArgNameDockerRegistry is the parameter name for docker registry to push to
	ArgNameDockerRegistry = "docker-registry"

	// ArgNameDockerUser is the parameter name for docker username used for pushing
	ArgNameDockerUser = "docker-user"

	// ArgNameDockerPW is the parameter name for docker password used for pushing
	ArgNameDockerPW = "docker-password"

	// ArgNameDockerBuildArg is the parameter name for build args passed in to docker. This parameter is repeatable.
	ArgNameDockerBuildArg = "docker-build-arg"

	// ArgNameDockerSecretBuildArg is the parameter name for build args passed in to docker. The argument value contains a secret which will be hidden from the log. This parameter is repeatable.
	ArgNameDockerSecretBuildArg = "docker-secret-build-arg"

	// ArgNameBuildEnv is the parameter name for build environment variables to be set. This parameter is repeatable
	ArgNameBuildEnv = "build-env"

	// ArgNamePull is the parameter determining if attempting to pull a newer version of the base images. Default: false
	ArgNamePull = "pull"

	// ArgNameNoCache is the parameter determining if not using any cached layer when building the image. Default: false
	ArgNameNoCache = "no-cache"

	// ArgNamePush is the parameter determining whether or not push should occur if the build would succeed. Default: false
	ArgNamePush = "push"

	// ArgNameDebug is the parameter that enables debug logs
	ArgNameDebug = "verbose"
)
