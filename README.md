# ACR builder

[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

## Build

Using Docker:

Execute the following commands from the root of the repository.

Linux:

```sh
$ docker build -f Dockerfile -t acb .
```

Windows:

```sh
$ docker build -f Windows.Dockerfile -t acb .
```

## Requirements

- Docker
- There are also dependency images that are used throughout the pipeline. Refer to the `baseimages` folder for corresponding Dockerfiles to generate these images, and review the list below for Linux/Windows.

## Linux Images

The following images are required:

- `scanner`
- `docker`
- `ubuntu`

## Windows Images

- `scanner`
- `docker`
- `microsoft/windowsservercore:1803`

## Usage

```sh
$ acb --help

Usage:
  acb [command]

Available Commands:
  build       Run a build
  exec        Execute a pipeline
  help        Help about any command
  init        Initialize a default template
  render      Render a template
  version     Print version information
```

## Building an image

See `acb build --help` for a list of all parameters.

Pushing to a registry:

```sh
$ docker run -v /var/run/docker.sock:/var/run/docker.sock acb build -t "foo:bar" -f "Dockerfile" --push -r foo.azurecr.io -u username -p password "https://github.com/Azure/acr-builder.git"
```

## Running a pipeline with a template

See `acb exec --help` for a list of all parameters.

```sh
$ docker run -v $(pwd):/workspace --workdir /workspace -v /var/run/docker.sock:/var/run/docker.sock acb exec --steps templating/testdata/helloworld/git-build.yaml --values templating/testdata/helloworld/values.yaml --id demo -r foo.azurecr.io -u username -p password
```
