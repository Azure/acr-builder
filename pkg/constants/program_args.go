package constants

// ArgNameBuildNumber is the parameter name for build number
const ArgNameBuildNumber = "build-number"

// ArgNameGitURL is the parameter name for git URL
const ArgNameGitURL = "git-url"

//ArgNameGitCloneTo is the parameter name for the git clone target directory
const ArgNameGitCloneTo = "git-clone-to"

// ArgNameGitBranch is the parameter name for git branch
const ArgNameGitBranch = "git-branch"

// ArgNameGitHeadRev is the parameter name for head revision
const ArgNameGitHeadRev = "git-head-revision"

// ArgNameGitPATokenUser is the parameter name for username for git username
const ArgNameGitPATokenUser = "git-username"

// ArgNameGitPAToken is the parameter name for git password
const ArgNameGitPAToken = "git-password"

// ArgNameGitXToken is the parameter name for github x-oath-basic token
const ArgNameGitXToken = "git-x-token"

// ArgNameLocalSource is the parameter name for local source directory
const ArgNameLocalSource = "local-source"

// ArgNameDockerfile is the parameter name for dockerfile
const ArgNameDockerfile = "docker-file"

// ArgNameDockerImage is the parameter name for docker image (registry url must be excluded from the image name parameter)
const ArgNameDockerImage = "docker-image"

// ArgNameDockerContextDir is the parameter name for docker context directory
const ArgNameDockerContextDir = "docker-context-dir"

// ArgNameDockerComposeFile is the parameter name for docker compose file used for build
const ArgNameDockerComposeFile = "docker-compose-file"

// ArgNameDockerComposeProjectDir is the parameter name for docker compose project directory
const ArgNameDockerComposeProjectDir = "docker-compose-project-dir"

// ArgNameDockerRegistry is the parameter name for docker registry to push to
const ArgNameDockerRegistry = "docker-registry"

// ArgNameDockerUser is the parameter name for docker username used for pushing
const ArgNameDockerUser = "docker-user"

// ArgNameDockerPW is the parameter name for docker password used for pushing
const ArgNameDockerPW = "docker-password"

// ArgNameDockerBuildArg is the parameter name for build args passed in to docker or docker-compose. This parameter is repeatable.
const ArgNameDockerBuildArg = "docker-build-arg"

// ArgNameBuildEnv is the parameter name for build environment variables to be set. This parameter is repeatable
const ArgNameBuildEnv = "build-env"

// ArgNamePush is the parameter determining whether or not push should occur if the build would succeed. Default: false
const ArgNamePush = "push"

// ArgNameDebug is the parameter that enables debug logs
const ArgNameDebug = "verbose"
