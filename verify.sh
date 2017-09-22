#!/bin/bash
set -e

## TODO: chmod me!!!

echo "Running Static Analysis tools..."

echo "Running GoVet..."
go vet $(go list ./... | grep -v /vendor/)

echo "Running ErrCheck..."
errcheck $(go list ./... | grep -v /vendor/)

echo "Running MegaCheck..."
megacheck $(go list ./... | grep -v /vendor/)

echo "Running golint..."
golint -set_exit_status $(go list ./... | grep -v '/vendor/' | grep -v '/mocks/' | grep -v '/constants')

echo "Running tests..."
go test -cover $(go list ./... | grep -v /vendor/ | grep -v '/tests/' | grep -v '/constants')

echo "Verification successful"
