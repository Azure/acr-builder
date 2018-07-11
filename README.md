# ACR builder

[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

## Build

Using Docker:

Execute the following commands from the root of the repository.

Linux:

```sh
$ docker build -f Dockerfile -t acb:linux .
```

Windows:

```sh
$ docker build -f Windows.Dockerfile -t acb:windows .
```

Using `make`:

```bash
$ make
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
  render      Render a template
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
$ acb exec --steps templating/testdata/helloworld/git-build.toml --values templating/testdata/helloworld/values.toml --id demo -r foo.azurecr.io -u username -p pw
```
