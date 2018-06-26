FROM golang:1.10-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make

FROM gobuild-base AS rally
WORKDIR /go/src/github.com/Azure/acr-builder
COPY . ./
RUN make static && mv rally /usr/bin/rally

FROM docker:18.03.1-ce-git
RUN apk add --no-cache \
    # Update Docker CLI config and set X-Meta-Source-Client header to ACR-BUILDER
    && mkdir -p ~/.docker \
    && echo '{"HttpHeaders":{"X-Meta-Source-Client":"ACR-BUILDER"}}' > ~/.docker/config.json
COPY --from=rally /usr/bin/rally /usr/bin/rally
ENTRYPOINT [ "rally" ]
CMD [ "--help" ]