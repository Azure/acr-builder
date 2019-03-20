# ACR builder

[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

ACR Builder is the backbone behind [Azure Container Registry Tasks](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-overview).

It can be used to automate container image patching and execute arbitrary containers for complex workflows.

You can find examples of how to create multi-step tasks [here](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-multi-step) and a reference to all of the available YAML properties [here](./docs/task.md).

## Requirements

- Docker
- There are also dependency images that are used throughout the task. Refer to the `baseimages` folder for corresponding Dockerfiles to generate these images, and review the list below for Linux/Windows.

## Building

With Docker, execute the following commands from the root of the repository.

Linux:

```sh
$ docker build -f Dockerfile -t acb .
```

Windows:

```sh
$ docker build -f Windows.Dockerfile -t acb .
```

## Linux Images

- `acb`
- `docker`
- `ubuntu`

## Windows Images

- `acb`
- `docker`
- `microsoft/windowsservercore:1803`

## Usage

```sh
$ acb --help

Usage:
  acb [command]

Available Commands:
  build       Run a build
  download    Download the specified context to a destination folder
  exec        Execute a task
  help        Help about any command
  render      Render a template
  scan        Scan a Dockerfile
  version     Print version information

Flags:
  -d, --debug   enable verbose output for debugging
  -h, --help    help for acb
```

## Building an image

See `acb build --help` for a list of all parameters.

```sh
$ docker run -v /var/run/docker.sock:/var/run/docker.sock acb build https://github.com/Azure/acr-builder.git
```

## Running a task

See `acb exec --help` for a list of all parameters.

```sh
$ docker run -v $(pwd):/workspace --workdir /workspace -v /var/run/docker.sock:/var/run/docker.sock acb exec --homevol $(pwd) -f templating/testdata/helloworld/git-build.yaml --values templating/testdata/helloworld/values.yaml --id demo -r foo.azurecr.io
```

## Rendering a template locally

```sh
$ acb render -f acb.yaml --values values.yaml
```

If your template uses `.Run.ID` or other `.Run` variables, refer to the full list of parameters using `acb render --help`.