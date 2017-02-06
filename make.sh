#!/bin/bash
set -e

check_prerequisites() {
  declare -a errors

  # Perform checks

  [[ "$GOPATH" == "" ]] && \
    errors=(${errors[@]} "GOPATH env missing")

  [[ -x "$GOPATH/bin/glide" ]] || \
    errors=("${errors[@]}" "glide not found in \"$GOPATH/bin/\" - see https://github.com/TrilliumIT/docker-vxlan-plugin/issues/2 for explanations on this need.")

  # Print status
  if [[ "${#errors[@]}" > 0 ]]; then
    echo "Errors:"
    for error in "${errors[@]}"; do
      echo "  $error"
    done
    return 1
  fi

  return 0
}

check_versions() {
  VERS="${LATEST_RELEASE}\n${MAIN_VER}"
  DKR_TAG="master"

  # For tagged commits
  if [ "$(git describe --tags)" = "$(git describe --tags --abbrev=0)" ] ; then
  	if [ $(printf ${VERS} | uniq | wc -l) -gt 1 ] ; then
  		echo "This is a release, all versions should match"
  		return 1
  	fi
  	DKR_TAG="latest"
  else
  	if [ $(printf ${VERS} | uniq | wc -l) -eq 1 ] ; then
  		echo "Please increment the version in main.go"
  		return 1
  	fi
  	if [ "$(printf ${VERS} | sort -V | tail -n 1)" != "${MAIN_VER}" ] ; then
  		echo "Please increment the version in main.go"
  		return 1
  	fi
  fi
}

LATEST_RELEASE=$(git describe --tags --abbrev=0 | sed "s/^v//g")
MAIN_VER=$(grep "version = " main.go | sed 's/[ \t]*version[ \t]*=[ \t]*//g' | sed 's/"//g')

check_prerequisites || exit 1
check_versions || exit 1

$GOPATH/bin/glide install

# Prepare build container (and build binary)
docker build -t trilliumit/docker-vxlan-plugin-build:v${MAIN_VER} -t trilliumit/docker-vxlan-plugin-build:${DKR_TAG} -f Dockerfile.build . || exit $?

# Retrieve binary
mkdir -p build
docker run -it --rm trilliumit/docker-vxlan-plugin-build:${DKR_TAG} cat /go/bin/docker-vxlan-plugin > build/docker-vxlan-plugin
chmod +x build/docker-vxlan-plugin

# Build final container
docker build -t trilliumit/docker-vxlan-plugin:v${MAIN_VER} -t trilliumit/docker-vxlan-plugin:${DKR_TAG} -f Dockerfile.run . || exit $?
