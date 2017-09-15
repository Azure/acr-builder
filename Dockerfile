FROM golang:1.8.3-alpine3.6 as build

RUN apk add --update --no-cache \
        openssl \
        ca-certificates \
        file \
        curl

ENV docker_compose_version 1.15.0
RUN curl -L --fail https://github.com/docker/compose/releases/download/$docker_compose_version/run.sh -o /usr/local/bin/docker-compose &&\
    chmod +x /usr/local/bin/docker-compose

WORKDIR /go/src/github.com/Azure/acr-builder
COPY ./ /go/src/github.com/Azure/acr-builder
RUN GOOS=linux GOARCH=amd64 go build

FROM docker:17.06.0-ce as output
RUN apk add --update --no-cache \
    git \
    openssh \
    openssl \
    ca-certificates

ENV GLIBC 2.23-r3
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/$GLIBC/glibc-$GLIBC.apk && \
    apk add --no-cache glibc-$GLIBC.apk && rm glibc-$GLIBC.apk && \
    ln -s /lib/libz.so.1 /usr/glibc-compat/lib/ && \
    ln -s /lib/libc.musl-x86_64.so.1 /usr/glibc-compat/lib && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/*

COPY --from=build /go/src/github.com/Azure/acr-builder/acr-builder /usr/local/bin
COPY --from=build /usr/local/bin/docker-compose /usr/local/bin
ENTRYPOINT ["/usr/local/bin/acr-builder"]
CMD []
