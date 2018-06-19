[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

# ACR builder

## Build

Run `make build`. Run `make help` for additional commands.

### Conditional or Optional parameters
* `--docker-registry` The docker registry to push to. This parameter will populate the `ACR_BUILD_DOCKER_REGISTRY` reserved environment variable (see `Build Environment`). Registry is required if the `--push` option is present.
* `--docker-user` The username for the docker registry specified above.
* `--docker-password` The password or token for registry specified above.
* `-f` Dockerfile to be used for building. If specified, the argument value can be a full path or a relative path to the source repository root. Otherwise, `Dockerfile` will be used.
* `-t` Image name to build to. It must be used alongside `--dockerfile` if push is required. Registry URL must be excluded from the image name parameter.
* `-c` Docker build context
* `--docker-build-arg` Build arguments to be passed to docker build. This parameter can be specified multiple times.
* `--docker-secret-build-arg` Build arguments to be passed to docker build. The argument value contains a secret which will be hidden from the log. This parameter can be specified multiple times.
* `--build-env` Custom environment variables defined for the build process. This parameter can be specified multiple times. (For more details, see `Build Environment`).
* `--push` Specify if push is required if build is successful.
* `--pull` Attempt to pull a newer version of the base images if it's already cached locally.
* `--isolation` Build using the specified isolation level
* `--no-cache` Not using any cached layer when building the image.
* `--verbose` Enable verbose output for debugging.