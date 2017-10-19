FROM golang:1.9.1-stretch as build
RUN go get -u github.com/kisielk/errcheck &&\
    go get -u honnef.co/go/tools/cmd/megacheck &&\
    go get -u github.com/golang/lint/golint

WORKDIR /go/src/github.com/Azure/acr-builder
COPY ./ /go/src/github.com/Azure/acr-builder
RUN echo "Running Static Analysis tools..." &&\
    echo "Running GoVet..." &&\
    go vet $(go list ./... | grep -v /vendor/) &&\
    echo "Running ErrCheck..." &&\
    errcheck $(go list ./... | grep -v /vendor/) &&\
    echo "Running MegaCheck..." &&\
    megacheck $(go list ./... | grep -v /vendor/) &&\
    echo "Running golint..." &&\
    golint -set_exit_status $(go list ./... | grep -v '/vendor/' | grep -v '/tests/') &&\
    echo "Running tests..." &&\
    go test -cover $(go list ./... | grep -v /vendor/ | grep -v '/tests/') &&\
    echo "Verification successful, building binaries..." &&\
    GOOS=linux GOARCH=amd64 go build

FROM docker/compose:1.16.1
RUN apk add --update --no-cache \
    docker \
    git \
    openssh \
    openssl \
    ca-certificates
COPY --from=build /go/src/github.com/Azure/acr-builder/acr-builder /usr/local/bin
ENTRYPOINT ["/usr/local/bin/acr-builder"]
CMD []
