# Scanner

## Build

Using Docker:

Execute the following commands from the root of the repository.

Linux:

```sh
$ docker build -f baseimages/scanner/Dockerfile -t scanner .
```

Windows:

```sh
$ docker build -f baseimages/scanner/Windows.Dockerfile -t scanner .
```

Using make:

```sh
$ make
```

### Examples

Scanning a local file:

```json
$ scanner scan -f Dockerfile . -t scanner

Dependencies:
[
    {
        "image": {
            "registry": "registry.hub.docker.com",
            "repository": "library/scanner",
            "tag": "latest",
            "digest": "",
            "reference": "scanner:latest"
        },
        "runtime-dependency": {
            "registry": "registry.hub.docker.com",
            "repository": "library/alpine",
            "tag": "latest",
            "digest": "",
            "reference": "alpine:latest"
        },
        "buildtime-dependency": [
            {
                "registry": "registry.hub.docker.com",
                "repository": "library/golang",
                "tag": "1.10-alpine",
                "digest": "",
                "reference": "golang:1.10-alpine"
            }
        ],
        "git": {
            "git-head-revision": ""
        }
    }
]
```

Scanning a git source:

```json
$ scanner scan -f Dockerfile https://github.com/Azure/acr-builder.git -t acr-builder

Dependencies:
[
    {
        "image": {
            "registry": "registry.hub.docker.com",
            "repository": "library/acr-builder",
            "tag": "latest",
            "digest": "",
            "reference": "acr-builder:latest"
        },
        "runtime-dependency": {
            "registry": "registry.hub.docker.com",
            "repository": "library/docker",
            "tag": "18.03.1-ce-git",
            "digest": "",
            "reference": "docker:18.03.1-ce-git"
        },
        "buildtime-dependency": [
            {
                "registry": "registry.hub.docker.com",
                "repository": "library/golang",
                "tag": "1.10.1-stretch",
                "digest": "",
                "reference": "golang:1.10.1-stretch"
            }
        ],
        "git": {
            "git-head-revision": "033ed90d4e0fa543c1910669dcf205578f957e85"
        }
    }
]
```

Scanning a tar:

```json
$ scanner scan -f "HelloWorld/Dockerfile" "https://acrbuild.blob.core.windows.net/public/aspnetcore-helloworld.tar.gz" -t hello-world

Dependencies:
[
    {
        "image": {
            "registry": "registry.hub.docker.com",
            "repository": "library/helloworld",
            "tag": "latest",
            "digest": "",
            "reference": "helloworld:latest"
        },
        "runtime-dependency": {
            "registry": "registry.hub.docker.com",
            "repository": "microsoft/aspnetcore",
            "tag": "2.0",
            "digest": "",
            "reference": "microsoft/aspnetcore:2.0"
        },
        "buildtime-dependency": [
            {
                "registry": "registry.hub.docker.com",
                "repository": "microsoft/aspnetcore-build",
                "tag": "2.0",
                "digest": "",
                "reference": "microsoft/aspnetcore-build:2.0"
            }
        ],
        "git": {
            "git-head-revision": ""
        }
    }
]
```