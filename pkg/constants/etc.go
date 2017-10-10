package constants

// ObfuscationString is the string is used to hide sensitive data such as token in logs
const ObfuscationString = "*************"

// StubDockerRegistry - docker-compose require a registry name to build. If push is required
// a stub value is used to allow the build to proceed
const StubDockerRegistry = "stub-registry"
