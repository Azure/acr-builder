package constants

const (
	acrBuildPrefix     = "ACR_BUILD_"
	dockerDomainPrefix = acrBuildPrefix + "DOCKER_"

	// ExportsDockerfilePath is the variable name for docker file path
	ExportsDockerfilePath = dockerDomainPrefix + "FILE"

	// ExportsDockerBuildContext is the docker build context directory
	ExportsDockerBuildContext = dockerDomainPrefix + "CONTEXT"

	// ExportsDockerRegistry is the docker registry to push to
	ExportsDockerRegistry = dockerDomainPrefix + "REGISTRY"

	// ExportsPushOnSuccess is the boolean value denoting whether the build will push on success
	ExportsPushOnSuccess = acrBuildPrefix + "PUSH_ON_SUCCESS"

	// ExportsBuildNumber is the current build number
	ExportsBuildNumber = acrBuildPrefix + "NUMBER"

	// ExportsBuildTimestamp is the timestamp when the build started in ISO format
	ExportsBuildTimestamp = acrBuildPrefix + "TIMESTAMP"
)
