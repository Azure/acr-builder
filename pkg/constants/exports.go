package constants

const (
	acrBuildPrefix     = "ACR_BUILD_"
	dockerDomainPrefix = acrBuildPrefix + "DOCKER_"
	gitDomainPrefix    = acrBuildPrefix + "GIT_"

	// ExportsWorkingDir is the variable name for checkout dir
	ExportsWorkingDir = acrBuildPrefix + "WORKING_DIR"

	// ExportsDockerfilePath is the variable name for docker file path
	ExportsDockerfilePath = dockerDomainPrefix + "FILE"

	// ExportsDockerBuildContext is the docker build context directory
	ExportsDockerBuildContext = dockerDomainPrefix + "CONTEXT"

	// ExportsDockerRegistry is the docker registry to push to
	ExportsDockerRegistry = dockerDomainPrefix + "REGISTRY"

	// ExportsGitSource is the current git source URL
	ExportsGitSource = gitDomainPrefix + "SOURCE"

	// ExportsGitBranch is current git branch
	ExportsGitBranch = gitDomainPrefix + "BRANCH"

	// ExportsGitHeadRev is the current git head revision
	ExportsGitHeadRev = gitDomainPrefix + "HEAD_REV"

	// ExportsGitAuthType is the git authentication type used
	ExportsGitAuthType = gitDomainPrefix + "AUTH_TYPE"

	// ExportsGitUser is the current git user
	ExportsGitUser = gitDomainPrefix + "USER"

	// ExportsPushOnSuccess is the boolean value denoting whether the build will push on success
	ExportsPushOnSuccess = acrBuildPrefix + "PUSH_ON_SUCCESS"

	// ExportsBuildNumber is the current build number
	ExportsBuildNumber = acrBuildPrefix + "NUMBER"

	// ExportsBuildTimestamp is the timestamp when the build started in ISO format
	ExportsBuildTimestamp = acrBuildPrefix + "TIMESTAMP"
)
