# Required.
# docker build -f baseimages/docker-cli/Dockerfile -t docker .
FROM docker:18.06.0-ce-git
# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN mkdir -p git-lfs && wget -qO- https://github.com/git-lfs/git-lfs/releases/download/v2.5.2/git-lfs-linux-amd64-v2.5.2.tar.gz | tar xz -C git-lfs; \
 	mv git-lfs/git-lfs /usr/bin/ && rm -rf git-lfs && git lfs install
