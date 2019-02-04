# Required.
# docker build -f baseimages/docker-cli/Dockerfile -t docker .
ARG DOCKER_CLI_BASE_IMAGE=docker:18.06.0-ce-git
FROM ${DOCKER_CLI_BASE_IMAGE}

ARG GIT_LFS_VERSION=2.5.2
# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN mkdir -p git-lfs && wget -qO- https://github.com/git-lfs/git-lfs/releases/download/v${GIT_LFS_VERSION}/git-lfs-linux-amd64-v${GIT_LFS_VERSION}.tar.gz | tar xz -C git-lfs; \
 	mv git-lfs/git-lfs /usr/bin/ && rm -rf git-lfs && git lfs install
