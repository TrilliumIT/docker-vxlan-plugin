FROM golang:1.5.3-wheezy

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>


ENV SRC_ROOT /go/src/github.com/clinta/docker-vxlan-plugin 

# Setup our directory and give convenient path via ln.
RUN mkdir -p ${SRC_ROOT}

WORKDIR ${SRC_ROOT}

# Used to only go get if sources change.
ADD . ${SRC_ROOT}/
RUN go get .
