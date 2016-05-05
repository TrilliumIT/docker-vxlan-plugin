#!/bin/bash
set -e

LATEST_TAG=$(git describe --tags)
grep "${LATEST_TAG}" Dockerfile
grep "${LATEST_TAG:1}" main.go

docker build -f ./Dockerbuild -t docker-vxlan-plugin-build . && docker run docker-vxlan-plugin-build cat /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
