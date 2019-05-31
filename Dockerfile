ARG DOCKER_CLI_BASE_IMAGE=docker:18.03.0-ce-git

FROM golang:1.12.5-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make

FROM gobuild-base AS acb
WORKDIR /go/src/github.com/Azure/acr-builder
COPY . .
RUN make binaries && mv bin/acb /usr/bin/acb

FROM ${DOCKER_CLI_BASE_IMAGE}

ARG GIT_LFS_VERSION=2.5.2
# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN mkdir -p git-lfs && wget -qO- https://github.com/git-lfs/git-lfs/releases/download/v${GIT_LFS_VERSION}/git-lfs-linux-amd64-v${GIT_LFS_VERSION}.tar.gz | tar xz -C git-lfs; \
 	mv git-lfs/git-lfs /usr/bin/ && rm -rf git-lfs && git lfs install

COPY --from=acb /usr/bin/acb /usr/bin/acb
ENTRYPOINT [ "acb" ]
CMD [ "--help" ]