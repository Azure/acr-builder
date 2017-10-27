## ACR builder

#### Build

Run `docker build --rm -t acr-builder .`. Note that acr-builder is intended to be used as a docker image.

#### Usage

##### Example

This project can be built using acr-builder itself, assuming you have run a valid acr builder image named `acr-builder`, running<br>
```./scripts/run-build.sh```<br>
Will rebuild the acr-docker image as `acr-builder`


Note that the file `./scripts/run-build.sh` is a convenience script and can be run on its own. You should be able to run it on its own.

Assuming you have the following:
* ACR builder image named `acr-builder`
* Your project has a `docker-compose.yml` or `Dockerfile` on its base directory
* You are currently authenticated to a target registry named `<registry>`

```
./run-build.sh --working-dir <source-dir> --push --docker-registry <registry>
```
On your project directory to build you project and push to desired registry.

##### Required Parameters


##### Conditional or Optional parameters
`--docker-registry` Docker registry to push to. This parameter will populate the `ACR_BUILD_DOCKER_REGISTRY` reserved environment variable (see `Build Environment`) Registry is required if `--push` options is present<br />
`--docker-user` Username for the docker registry specified above<br />
`--docker-password` Password or token for registry specified above<br />
`--archive` The URL of a tar.gz archive which contains the source code<br />
`--git-url` Git url to the project. Clone operation will be ignored if `--git-clone-to` folder exist and is not empty and this parameter will not be required<br />
`--git-branch` The git branch to checkout. If it is not given, no checkout command would be performed<br />
`--git-head-revision` Desired git HEAD revision, note that providing this parameter will cause the branch parameter to be ignored<br />
`--git-username` Git username<br />
`--git-password` Git password<br />
`--working-dir` Working directory for the build.<br />
`--docker-file` Dockerfile to be used for building<br />
`--docker-image` Image name to build to. It must be used alongside `--dockerfile` if push is required. Registry url must be excluded from the image name parameter.<br />
`--docker-context-dir` Docker build context. Optional, to be used alongside `--dockerfile`<br />
`--docker-compose-file` Docker Compose file to be invoked for build and push<br />
`--docker-build-arg` Build arguments to be passed to docker build or docker-compose build. This parameter can be specified multiple times.<br />
`--build-env` Custom environment variables defined for the build process. This parameter can be specified multiple times. (For more details, see `Build Environment`)<br />
`--push` Specify if push is required if build is successful<br />
`--verbose` Enable verbose output for debugging<br />

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
    image: ${ACR_BUILD_DOCKER_REGISTRY}hello-builder-1.0-${ACR_BUILD_BUILD_NUMBER}

  hello-multistage:
    build: ./hello-multistage
    image: ${ACR_BUILD_DOCKER_REGISTRY}hello-multistage:${ACR_BUILD_BUILD_NUMBER}
```
