FROM golang as build
ADD . /go/src/github.com/Azure/acr-builder
ENV docker_compose_version 1.15.0
RUN mkdir /artifacts && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /artifacts/acr-build /go/src/github.com/Azure/acr-builder/execution/main/program.go
RUN curl -L https://github.com/docker/compose/releases/download/${docker_compose_version}/docker-compose-`uname -s`-`uname -m` > /usr/local/bin/docker-compose

FROM docker:git
COPY --from=0 /artifacts/acr-build .
COPY --from=0 /usr/local/bin/docker-compose /usr/local/bin
ENV GLIBC 2.23-r3
RUN apk update && apk add --no-cache openssl ca-certificates && \
    wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/$GLIBC/glibc-$GLIBC.apk && \
    apk add --no-cache glibc-$GLIBC.apk && rm glibc-$GLIBC.apk && \
    ln -s /lib/libz.so.1 /usr/glibc-compat/lib/ && \
    ln -s /lib/libc.musl-x86_64.so.1 /usr/glibc-compat/lib && \
    apk --update add git openssh && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/* && \
    chmod +x /usr/local/bin/docker-compose
ENTRYPOINT ["/acr-build"]
CMD []
