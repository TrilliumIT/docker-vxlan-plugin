# Does not work from scratch,
# Need to determine the missing dependencies
#FROM scratch

FROM alpine:3.5

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

ADD ./build /

ENTRYPOINT ["/docker-vxlan-plugin"]
