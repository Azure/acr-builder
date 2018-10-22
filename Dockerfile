FROM golang:1.10-alpine AS gobuild-base
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
COPY --from=acb /usr/bin/acb /usr/bin/acb
ENTRYPOINT [ "acb" ]
CMD [ "--help" ]