# ACR builder

| Linux Build | Windows Build | Go Report |
|---|---|---|
|[![Build Status](https://dev.azure.com/azurecontainerregistry/acr-builder/_apis/build/status/acr-builder_linux?branchName=main)](https://dev.azure.com/azurecontainerregistry/acr-builder/_build/latest?definitionId=18&branchName=main)|[![Build Status](https://dev.azure.com/azurecontainerregistry/acr-builder/_apis/build/status/acr-builder_windows?branchName=main)](https://dev.azure.com/azurecontainerregistry/acr-builder/_build/latest?definitionId=19&branchName=main)|[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/acr-builder)](https://goreportcard.com/report/github.com/Azure/acr-builder)|

ACR Builder is the backbone behind [Azure Container Registry Tasks](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-overview).

It can be used to automate container image patching and execute arbitrary containers for complex workflows.

You can find examples of how to create multi-step tasks [here](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-multi-step).

## Task Schema

For a list of all available YAML properties, please review the [Task schema](./docs/task.md).

## Templating

To understand templating and how to provide custom values to your runs, review [templates](./docs/templates.md).

## Requirements

- Docker

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

## Usage

```sh
$ acb --help

NAME:
   acb - run and build containers on Azure Container Registry

USAGE:
   acb [global options] command [command options] [arguments...]

VERSION:
   38f06e5

COMMANDS:
     build      build container images
     download   download the specified context to a destination folder
     exec       execute a task file
     render     render the specified template
     scan       scan a Dockerfile for dependencies
     version    print the client and runtime versions
     getsecret  gets the secret value from a specified vault
     help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
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


## F5 experience on VSCode

You can install `delve`, and add something like this to your `.vscode/launch.json` file - and hit f5. The binary executes from under `./cmd/acb`, so you can put any Task files that you want to debug.

First, you'd have to run a few commands:

### Create a source volume for your workspace (i.e. your context, Dockerfiles, Task yaml files)
```sh
sudo docker volume create source
sudo docker volume inspect source
[
    {
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Driver": "local",
        "Labels": {},
        "Mountpoint": "/var/lib/docker/volumes/source/_data",
        "Name": "source",
        "Options": {},
        "Scope": "local"
    }
]
sudo rm -rf /var/lib/docker/volumes/source/_data
sudo ln -s $(pwd) /var/lib/docker/volumes/source/_data
```

Now, you can add your Dockerfiles or Task files to `cmd/acb/` folder.

If your testing Task file contains pulling/pushing stuff off a private repository, then you will have to do the following step. Make sure you are logged in to the repo using `docker login`.
If you don't need that, you can skip the following step.

### Create a home volume for Docker to find your registry credentials.

From the `cmd/acb/` folder run

```sh
sudo docker volume create home
sudo docker volume inspect home
[
    {
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Driver": "local",
        "Labels": {},
        "Mountpoint": "/var/lib/docker/volumes/home/_data",
        "Name": "home",
        "Options": {},
        "Scope": "local"
    }
]
sudo rm -rf /var/lib/docker/volumes/home/_data
sudo ln -s $(HOME)/.docker /var/lib/docker/volumes/home/_data
```

### Create `launch.json` file in your `.vscode` folder:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceRoot}/cmd/acb",
            "env": {},
            "args": [
               "exec",
               "--homevol",
               "source",
               "-f",
               "./test.acb.yml",
               ".",
                "--id",
                "blah",
                "--registry",
                "samashah.azurecr.io"
            ]
        }
    ]
}
```

Press F5.
