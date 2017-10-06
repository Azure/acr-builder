## ACR builder

#### Build

Run `docker build --rm -t acr-builder .`. Note that acr-builder is intended to be used as a docker image.

#### Usage

##### Example

This project can be built using acr-builder itself, assuming you have run a valid acr builder image named `acr-builder`, running
`./scripts/run.sh --docker-image acr-builder`
Will rebuild the acr-docker image

##### Required Parameters
`--docker-registry` Docker registry to push to. This parameter will populate the `ACR_BUILD_DOCKER_REGISTRY` reserved environment variable (see `Build Environment`)<br />

##### Conditional or Optional parameters
`--docker-user` Username for the docker registry specified above<br />
`--docker-secret` Password or token for registry specified above<br />
`--git-url` Git url to the project. Clone operation will be ignored if `--git-clone-to` folder exist and is not empty and this parameter will not be required<br />
`--git-clone-to` Directory to clone to. If the directory exists, we won't clone again and will just clean and pull the directory. The default value is `$HOME/acr-builder/src`<br />
`--git-branch` The git branch to checkout. If it is not given, no checkout command would be performed<br />
`--git-head-revision` Desired git HEAD revision, note that providing this parameter will cause the branch parameter to be ignored<br />
`--git-username` Git username<br />
`--git-password` Git password<br />
`--local-source` Local source directory. Specifying this parameter tells the builder no source control is used and it would use the specified directory as source<br />
`--dockerfile` Dockerfile to be used for building<br />
`--docker-image` Image name to build to. It must be used alongside `--dockerfile` if push is required. Registry url must be excluded from the image name parameter.<br />
`--docker-context-dir` Docker build context. Optional, to be used alongside `--dockerfile`<br />
`--docker-compose-file` Docker Compose file to be invoked for build and push<br />
`--dockerfile` Dockerfile to be used for building<br />
`--docker-build-arg` Build arguments to be passed to docker build or docker-compose build. This parameter can be specified multiple times.<br />
`--build-env` Custom environment variables defined for the build process. This parameter can be specified multiple times. (For more details, see `Build Environment`)<br />
`--push` Specify if push is required if build is successful<br />

##### Build Environment
By setting environment variable with parameter `--build-env <VAR_NAME>=<VAR_VALUE>`, the builder would be aware of the environmental variables throughout the build and the environment will be set for all child processes. ACR builder has a set of reserved environment variables such as `ACR_BUILD_BUILD_NUMBER` and `ACR_BUILD_DOCKER_REGISTRY` mentioned in the parameters paragraph. The user will set them by passing in the optional parameters `--build-number` and `--docker-registry` and they cannot be override with `--build-env`

Furthermore, ACR builder also populates the following variables during build so the child process can make use of these values:

`ACR_BUILD_NUMBER` Current build number<br />
`ACR_BUILD_TIMESTAMP` Timestamp of where the build start in ISO format<br />
`ACR_BUILD_SOURCE_DIR` Source directory on the current build system<br />
`ACR_BUILD_DOCKER_COMPOSE_FILE` Current docker compose file being used, relative to the source directory<br />
`ACR_BUILD_GIT_URL` Git URL<br />
`ACR_BUILD_GIT_BRANCH` Current Git branch<br />
`ACR_BUILD_GIT_HEAD_REV` SHA for current Git Head Revision<br />
`ACR_BUILD_PUSH_IMAGES` Indicate whether current build will push on success

##### Compose File
In `docker-compose.yml`, Build image should be prefixed by the reserved environmental variable `ACR_BUILD_DOCKER_REGISTRY` so they are pushed to the desired registry. You can also use the reserved `ACR_BUILD_BUILD_NUMBER` to postfix your image or tag
```yaml
version: '2'
services:
  hello:
    build: ./hello-builder
    image: ${ACR_BUILD_DOCKER_REGISTRY}/hello-builder-1.0-${ACR_BUILD_BUILD_NUMBER}

  hello-multistage:
    build: ./hello-multistage
    image: ${ACR_BUILD_DOCKER_REGISTRY}/hello-multistage:${ACR_BUILD_BUILD_NUMBER}
```