[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

# ACR builder

## Building from source

Run `make build`. Use `make help` to discover additional commands.

## CLI

`rally --help` can be used to see all available commands.

## Building an image

See `rally build --help` for a list of all parameters.

Pushing to a registry:

`rally build -t "foo:bar" -c "https://github.com/Azure/acr-builder.git" -f "Dockerfile" --push -r foo.azurecr.io -u foo -p foo`

### Conditional or Optional parameters
* `--registry` The docker registry to push to. This parameter will populate the `ACR_BUILD_DOCKER_REGISTRY` reserved environment variable (see `Build Environment`). Registry is required if the `--push` option is present.
* `--username` The username for the docker registry specified above.
* `--password` The password or token for registry specified above.
* `-f` Dockerfile to be used for building. If specified, the argument value can be a full path or a relative path to the source repository root. Otherwise, `Dockerfile` will be used.
* `-t` Image name to build to. It must be used alongside `--dockerfile` if push is required. Registry URL must be excluded from the image name parameter.
* `-c` Docker build context
* `--build-arg` Build arguments to be passed to docker build. This parameter can be specified multiple times.
* `--secret-build-arg` Build arguments to be passed to docker build. The argument value contains a secret which will be hidden from the log. This parameter can be specified multiple times.
* `--push` Specify if push is required if build is successful.
* `--pull` Attempt to pull a newer version of the base images if it's already cached locally.
* `--isolation` Build using the specified isolation level
* `--no-cache` Not using any cached layer when building the image.
* `--debug` Enable verbose output for debugging.

## Running a pipeline with a template

See `rally exec --help` for a list of all parameters.

```
rally exec --steps helloworld.toml --template-path templating/testdata/helloworld --id demo -r foo.azurecr.io -u foo -p foo --debug
```