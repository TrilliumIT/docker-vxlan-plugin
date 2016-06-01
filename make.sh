#!/bin/bash
set -e

LATEST_RELEASE=$(git describe --tags --abbrev=0 | sed "s/^v//g")
MAIN_VER=$(grep "version = " main.go | sed 's/[ \t]*version[ \t]*=[ \t]*//g' | sed 's/"//g')
VERS="${LATEST_RELEASE}\n${DOCKER_VER}\n${MAIN_VER}"

# For tagged commits
if [ "$(git describe --tags)" = "$(git describe --tags --abbrev=0)" ] ; then
	if [ $(printf ${VERS} | uniq | wc -l) -gt 1 ] ; then
		echo "This is a release, all versions should match"
		exit 1
	fi
	DKR_TAG="latest"
else
	if [ $(printf ${VERS} | uniq | wc -l) -eq 1 ] ; then
		echo "Please increment the version in main.go"
		exit 1
	fi
	if [ "$(printf ${VERS} | sort -V | tail -n 1)" != "${MAIN_VER}" ] ; then
		echo "Please increment the version in main.go"
		exit 1
	fi
	DKR_TAG="master"
fi

docker build -t trilliumit/docker-vxlan-plugin:v${MAIN_VER} -t trilliumit/docker-vxlan-plugin:${DKR_TAG} . || exit $?

docker run -it --rm --entrypoint cat trilliumit/docker-vxlan-plugin:master /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
