#!/bin/bash

set -e

scriptPath=$(cd $(dirname "$0") && pwd)

image=minixxie/apollo-server-go
commitID=$(git rev-parse HEAD)
tag=0.0.4
platforms=linux/amd64,linux/arm64/v8

nerdctl build . \
	-f Containerfile --platform $platforms \
	--tag $image:$tag \
	--namespace=k8s.io
nerdctl tag $image:$tag $image:latest --namespace=k8s.io
nerdctl images --namespace=k8s.io | grep $image
nerdctl login
nerdctl push --platform $platforms $image:$tag --namespace=k8s.io
nerdctl push --platform $platforms $image:latest --namespace=k8s.io
