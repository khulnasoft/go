#!/bin/bash
set -ex
cd $(dirname "${BASH_SOURCE[0]}")

# Build image
VERSION=$(printf "%05d" $BUILDKITE_BUILD_NUMBER)_$(date +%Y-%m-%d)_$(git rev-parse --short HEAD)
docker build -t khulnasoft/lang-go:$VERSION .

# Upload to Docker Hub
docker push khulnasoft/lang-go:$VERSION
docker tag khulnasoft/lang-go:$VERSION khulnasoft/lang-go:latest
docker push khulnasoft/lang-go:latest
docker tag khulnasoft/lang-go:$VERSION khulnasoft/lang-go:insiders
docker push khulnasoft/lang-go:insiders
