FROM golang:alpine as Build
ADD . /src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /artifacts/hello-multistage /src/program.go

FROM alpine
COPY --from=0 /artifacts/hello-multistage .
ENTRYPOINT ["/hello-multistage"]
CMD []

