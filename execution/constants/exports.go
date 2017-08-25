package constants

const acrBuildPrefix = "ACR_BUILD_"
const CheckoutDirVar = acrBuildPrefix + "CHECKOUT_DIR"

const dockerDomainPrefix = acrBuildPrefix + "DOCKER_"
const DockerfilePathVar = dockerDomainPrefix + "FILE"
const DockerBuildContextVar = dockerDomainPrefix + "CONTEXT"
const DockerPushImageVar = dockerDomainPrefix + "PUSH_TO"
const DockerRegistryVar = dockerDomainPrefix + "REGISTRY"

const dockerComposeDomainPrefix = acrBuildPrefix + "DOCKER_COMPOSE_"
const DockerComposeFileVar = dockerComposeDomainPrefix + "FILE"

const gitDomainPrefix = acrBuildPrefix + "GIT_"
const GitSourceVar = gitDomainPrefix + "SOURCE"
const GitBranchVar = gitDomainPrefix + "BRANCH"
const GitAuthTypeVar = gitDomainPrefix + "AUTH_TYPE"
const GitUserVar = gitDomainPrefix + "USER"

const PushOnSuccessVar = acrBuildPrefix + "PUSH_ON_SUCCESS"
const BuildNumberVar = acrBuildPrefix + "BUILD_NUMBER"
