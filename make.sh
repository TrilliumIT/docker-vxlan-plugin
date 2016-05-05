#!/bin/bash
set -e

LATEST_RELEASE=$(git describe --tags --apprev=0 | sed "s/^v//g")
DOCKER_VER=$(grep "ENV VER=" Dockerfile | sed "s/^ENV VER=v//g")
MAIN_VER=$(grep "version = " main.go | sed 's/[ \t]*version[ \t]*=[ \t]*//g' | sed 's/"//g')
VERS=$(echo "${LATEST_RELEASE}\n${DOCKER_VER}\n${MAIN_VER}")

# For tagged commits
if [ $(git describe --tags) = ${LATEST_RELEASE} ] ; then
	if [ $(echo ${VERS} | uniq | wc -l) -gt 1 ]
		echo "This is a release and the versions don't match"
		exit 1
	fi
fi

if [ $(echo ${VERS} | sort -V | head -n l) != ${LATEST_RELEASE} ] ; then
	echo "Current versions are less than the latest release"
	exit 1
fi

docker build -f ./Dockerbuild -t docker-vxlan-plugin-build . && docker run docker-vxlan-plugin-build cat /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
