ARG DOCKER_CLI_BASE_IMAGE=mcr.microsoft.com/acr/moby-cli:linux-latest

FROM mcr.microsoft.com/oss/go/microsoft/golang:1.21-fips-cbl-mariner2.0 AS gobuild-base

FROM gobuild-base AS acb
RUN tdnf update -y && tdnf install make git -y
WORKDIR /go/src/github.com/Azure/acr-builder
COPY . .
RUN make binaries && mv bin/acb /usr/bin/acb

FROM ${DOCKER_CLI_BASE_IMAGE}

ARG GIT_LFS_VERSION=2.5.2
# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN mkdir -p git-lfs && curl -sL https://github.com/git-lfs/git-lfs/releases/download/v${GIT_LFS_VERSION}/git-lfs-linux-amd64-v${GIT_LFS_VERSION}.tar.gz | tar xz -C git-lfs; \
	mv git-lfs/git-lfs /usr/bin/ && rm -rf git-lfs && git lfs install --system

COPY --from=acb /usr/bin/acb /usr/bin/acb
ENTRYPOINT [ "acb" ]
CMD [ "--help" ]