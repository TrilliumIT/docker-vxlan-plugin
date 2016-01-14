FROM ubuntu:14.04

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

ADD docker-vxlan-plugin /

VOLUME ["/var/run/docker.sock:/var/run/docker.sock", "/run/docker/plugins/docker-vxlan-plugin:/run/docker/plugins/docker-vxlan-plugin"]

ENTRYPOINT ["/docker-vxlan-plugin"]
