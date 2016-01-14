FROM ubuntu:14.04

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

ADD docker-vxlan-plugin /
CMD ["/docker-vxlan-plugin"]
