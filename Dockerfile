FROM golang:1.5.4-wheezy

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

ENV GO15VENDOREXPERIMENT 1

RUN go get github.com/Masterminds/glide

ENV SRC_ROOT /go/src/github.com/TrilliumIT/docker-vxlan-plugin 

# Setup our directory and give convenient path via ln.
RUN mkdir -p ${SRC_ROOT}

WORKDIR ${SRC_ROOT}

# Used to only go get if sources change.
ADD . ${SRC_ROOT}/
RUN go get -t $($GOPATH/bin/glide novendor)

ENTRYPOINT ["/go/bin/docker-vxlan-plugin"]
