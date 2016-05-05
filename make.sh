#!/bin/bash
set -e

LATEST_RELEASE=$(git describe --tags --abbrev=0 | sed "s/^v//g")
DOCKER_VER=$(grep "ENV VER=" Dockerfile | sed "s/^ENV VER=v//g")
MAIN_VER=$(grep "version = " main.go | sed 's/[ \t]*version[ \t]*=[ \t]*//g' | sed 's/"//g')
VERS="${LATEST_RELEASE}\n${DOCKER_VER}\n${MAIN_VER}"

# Dockerfile should always match latest release
if [ "${DOCKER_VER}" != "${LATEST_RELEASE}" ] ; then
	echo "Dockerfile does not match latest tag"
	exit 1
fi

# For tagged commits
if [ "$(git describe --tags)" = "${LATEST_RELEASE}" ] ; then
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
	DKR_TAG="prerelease"
fi


docker build -f ./Dockerbuild -t clinta/docker-vxlan-plugin-build:v${MAIN_VER} . && docker run clinta/docker-vxlan-plugin-build:v${MAIN_VER} cat /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
docker build -f ./Dockerlocal -t clinta/docker-vxlan-plugin:v${MAIN_VER} -t clinta/docker-vxlan-plugin:${DKR_TAG} .
