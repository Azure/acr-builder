FROM golang:1.11.2-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make

FROM gobuild-base AS acb
WORKDIR /go/src/github.com/Azure/acr-builder
COPY . .
RUN make static && mv acb /usr/bin/acb

FROM docker:18.03.0-ce-git

# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN mkdir -p git-lfs && wget -qO- https://github.com/git-lfs/git-lfs/releases/download/v2.5.2/git-lfs-linux-amd64-v2.5.2.tar.gz | tar xz -C git-lfs; \
 	mv git-lfs/git-lfs /usr/bin/ && rm -rf git-lfs && git lfs install

COPY --from=acb /usr/bin/acb /usr/bin/acb
ENTRYPOINT [ "acb" ]
CMD [ "--help" ]