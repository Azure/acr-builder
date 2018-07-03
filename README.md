# ACR builder

[![Build Status](https://travis-ci.org/Azure/acr-builder.svg?branch=master)](https://travis-ci.org/Azure/acr-builder)

## Building from source

Run `make build`. Use `make help` to discover additional commands.

## Requirements

- Docker
- In order to run `build`, you must create an image called scanner. This image can be built using `docker build -f baseimages/scanner/Dockerfile -t scanner .` at the root of this repository.

## CLI

`acb --help` can be used to see all available commands.

## Building an image

See `acb build --help` for a list of all parameters.

Pushing to a registry:

`acb build -t "foo:bar" -f "Dockerfile" --push -r foo.azurecr.io -u foo -p foo "https://github.com/Azure/acr-builder.git"`

## Running a pipeline with a template

See `acb exec --help` for a list of all parameters.

```bash
acb exec --steps helloworld.toml --template-path templating/testdata/helloworld --id demo -r foo.azurecr.io -u foo -p foo --debug
```