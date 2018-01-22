[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

# ACR builder

## Build

Run `docker build --rm -t acr-builder .`. Note that acr-builder is intended to be used as a docker image.

### Example

This project can be built using acr-builder itself, assuming you have a valid acr-builder image named `acr-builder`. Running `./scripts/run-build.sh` will rebuild the acr-docker image as `acr-builder`.

Note that the file `./scripts/run-build.sh` is a convenience script and can be run on its own. You should be able to run it on its own.

Assuming you have the following:
* ACR builder image named `acr-builder`
* Your project has a `docker-compose.yml` or `Dockerfile` on its base directory
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
* `--archive` The URL of a tar.gz archive which contains the source code.
* `--git-url` The Git URL to the project. Clone operation will be ignored if the `--git-clone-to` folder exists and is not empty. In that case, this parameter will not be required.
* `--git-branch` The Git branch to checkout. If it is not given, no checkout command would be performed.
* `--git-head-revision` Desired Git HEAD revision, note that providing this parameter will cause the branch parameter to be ignored.
* `--git-username` Git username.
* `--git-password` Git password.
* `--working-dir` Working directory for the build.
* `--docker-file` Dockerfile to be used for building.
* `--docker-image` Image name to build to. It must be used alongside `--dockerfile` if push is required. Registry URL must be excluded from the image name parameter.
* `--docker-context-dir` Docker build context. Optional, to be used alongside `--dockerfile`.
* `--docker-compose-file` Docker Compose file to be invoked for build and push.
* `--docker-build-arg` Build arguments to be passed to docker build or docker-compose build. This parameter can be specified multiple times.
* `--build-env` Custom environment variables defined for the build process. This parameter can be specified multiple times. (For more details, see `Build Environment`).
* `--push` Specify if push is required if build is successful.
* `--verbose` Enable verbose output for debugging.

### Build Environment
You can set an environment variable with `--build-env <VAR_NAME>=<VAR_VALUE>` and the builder will be aware of an environment variable throughout the build (the same goes for all child processes). ACR builder has a set of reserved environment variables such as `ACR_BUILD_BUILD_NUMBER` and `ACR_BUILD_DOCKER_REGISTRY` mentioned in the parameters paragraph. You can set them by passing in the optional parameters `--build-number` and `--docker-registry` and they can't be overridden with `--build-env`.

Furthermore, ACR builder also populates the following variables during build so the child process can make use of these values:

* `ACR_BUILD_NUMBER` Current build number.
* `ACR_BUILD_TIMESTAMP` Timestamp of where the build start in ISO format.
* `ACR_BUILD_SOURCE_DIR` Source directory on the current build system.
* `ACR_BUILD_DOCKER_COMPOSE_FILE` Current docker compose file being used, relative to the source directory.
* `ACR_BUILD_GIT_URL` Git URL.
* `ACR_BUILD_GIT_BRANCH` Current Git branch.
* `ACR_BUILD_GIT_HEAD_REV` SHA for current Git Head Revision.
* `ACR_BUILD_PUSH_IMAGES` Indicates whether or not the current build will push on success.

### Compose File
In `docker-compose.yml`, the build image should be prefixed by the reserved environmental variable `ACR_BUILD_DOCKER_REGISTRY` so that it's pushed to the desired registry. You can also use the reserved `ACR_BUILD_BUILD_NUMBER` to postfix your image or tag.

```yaml
version: '2'
services:
  hello:
    build: ./hello-builder
    image: ${ACR_BUILD_DOCKER_REGISTRY}hello-builder-1.0-${ACR_BUILD_BUILD_NUMBER}

  hello-multistage:
    build: ./hello-multistage
    image: ${ACR_BUILD_DOCKER_REGISTRY}hello-multistage:${ACR_BUILD_BUILD_NUMBER}
```
