[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

# ACR builder

## Build

Run `docker build --rm -t acr-builder .`. Note that acr-builder is intended to be used as a docker image.

### Example

This project can be built using acr-builder itself, assuming you have a valid acr-builder image named `acr-builder`. Running `./scripts/run-build.sh` will rebuild the acr-docker image as `acr-builder`.

Note that the file `./scripts/run-build.sh` is a convenience script and can be run on its own. You should be able to run it on its own.

Assuming you have the following:
* ACR builder image named `acr-builder`
* Your project has a `Dockerfile` on its base directory
* You are currently authenticated to a target registry named `<registry>`

Run the following on your project directory to build the project and push to a desired registry:
```
./run-build.sh --working-dir <source-dir> --push --docker-registry <registry>
```

### Required Parameters
- N/A for now.

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
* `--hyperv-isolation` Build using Hyper-V hypervisor partition based isolation. This is used for Windows container builds.
* `--no-cache` Not using any cached layer when building the image.
* `--verbose` Enable verbose output for debugging.

### Build Environment
You can set an environment variable with `--build-env <VAR_NAME>=<VAR_VALUE>` and the builder will be aware of an environment variable throughout the build (the same goes for all child processes). ACR builder has a set of reserved environment variables such as `ACR_BUILD_DOCKER_REGISTRY` mentioned in the parameters paragraph. You can set them by passing in the optional parameter `--docker-registry` and they can't be overridden with `--build-env`.

Furthermore, ACR builder also populates the following variables during build so the child process can make use of these values:

* `ACR_BUILD_TIMESTAMP` Timestamp of where the build start in ISO format.
* `ACR_BUILD_PUSH_IMAGES` Indicates whether or not the current build will push on success.
