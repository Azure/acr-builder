# ACR builder

[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

## Building from source

Build using `make`:

```bash
$ make build

go build -tags "" -ldflags "-w -X acb/version.GITCOMMIT=4fb5952-dirty -X acb/version.VERSION=v1.0.0" -o acb .
```

For additional commands, try `make help`.

## Requirements

- Docker
- In order to run `build`, you must create an image called scanner. This image can be built using `docker build -f baseimages/scanner/Dockerfile -t scanner .` at the root of this repository.

## CLI

```bash
$ acb --help

Usage:
  acb [command]

Available Commands:
  build       Run a build
  exec        Execute a pipeline
  help        Help about any command
  init        Initialize a default template
  lint        Lint a template
  version     Print version information
```

## Building an image

See `acb build --help` for a list of all parameters.

Pushing to a registry:

```bash
$ acb build -t "foo:bar" -f "Dockerfile" --push -r foo.azurecr.io -u foo -p foo "https://github.com/Azure/acr-builder.git"
```

## Running a pipeline with a template

See `acb exec --help` for a list of all parameters.

```bash
$ ./acb exec --steps templating/testdata/helloworld/git-build.toml --values templating/testdata/helloworld/values.toml --id ericdemo --debug -r foo.azurecr.io -u username -p pw
```