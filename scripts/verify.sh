#!/bin/bash
set -e

echo "Running Static Analysis tools..."

echo "Running GoVet..."
go vet $(go list ./... | grep -v /vendor/)

echo "Running ErrCheck..."
errcheck $(go list ./... | grep -v /vendor/)

echo "Running MegaCheck..."
megacheck $(go list ./... | grep -v /vendor/)

echo "Running golint..."
golint -set_exit_status $(go list ./... | grep -v '/vendor/' | grep -v '/tests/')

echo "Running tests..."
go test -cover $(go list ./... | grep -v /vendor/ | grep -v '/tests/')

echo "Verification successful"
