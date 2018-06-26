FROM golang:1.10-alpine AS gobuild-base
RUN apk add --no-cache \
	git \
	make
FROM gobuild-base AS scanner
COPY . /go/src/github.com/Azure/acr-builder
WORKDIR /go/src/github.com/Azure/acr-builder/baseimages/scanner
RUN make static && mv scanner /usr/bin/scanner

FROM alpine
RUN apk add --no-cache \
	git
COPY --from=scanner /usr/bin/scanner /usr/bin/scanner
ENTRYPOINT [ "scanner" ]
CMD [ "--help" ]