package constants

const acrBuildPrefix = "ACR_BUILD_"
const dockerDomainPrefix = acrBuildPrefix + "DOCKER_"
const gitDomainPrefix = acrBuildPrefix + "GIT_"
const dockerComposeDomainPrefix = acrBuildPrefix + "DOCKER_COMPOSE_"

// ExportsWorkingDir is the variable name for checkout dir
const ExportsWorkingDir = acrBuildPrefix + "WORKING_DIR"

// ExportsDockerfilePath is the variable name for docker file path
const ExportsDockerfilePath = dockerDomainPrefix + "FILE"

// ExportsDockerBuildContext is the docker build context directory
const ExportsDockerBuildContext = dockerDomainPrefix + "CONTEXT"

// ExportsDockerPushImage is the image name to push to
const ExportsDockerPushImage = dockerDomainPrefix + "PUSH_TO"

// ExportsDockerRegistry is the docker registry to push to
const ExportsDockerRegistry = dockerDomainPrefix + "REGISTRY"

// ExportsDockerComposeFile is the docker compose file used for build and push
const ExportsDockerComposeFile = dockerComposeDomainPrefix + "FILE"

// ExportsGitSource is the current git source URL
const ExportsGitSource = gitDomainPrefix + "SOURCE"

// ExportsGitBranch is current git branch
const ExportsGitBranch = gitDomainPrefix + "BRANCH"

// ExportsGitHeadRev is the current git head revision
const ExportsGitHeadRev = gitDomainPrefix + "HEAD_REV"

// ExportsGitAuthType is the git authentication type used
const ExportsGitAuthType = gitDomainPrefix + "AUTH_TYPE"

// ExportsGitUser is the current git user
const ExportsGitUser = gitDomainPrefix + "USER"

// ExportsPushOnSuccess is the boolean value denoting whether the build will push on success
const ExportsPushOnSuccess = acrBuildPrefix + "PUSH_ON_SUCCESS"

// ExportsBuildNumber is the current build number
const ExportsBuildNumber = acrBuildPrefix + "NUMBER"

// ExportsBuildTimestamp is the timestamp when the build started in ISO format
const ExportsBuildTimestamp = acrBuildPrefix + "TIMESTAMP"
