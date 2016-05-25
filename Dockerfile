FROM debian:jessie
ENV VER=v0.5.1

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

ADD https://github.com/clinta/docker-vxlan-plugin/releases/download/${VER}/docker-vxlan-plugin /docker-vxlan-plugin
RUN chmod +x /docker-vxlan-plugin

ENTRYPOINT ["/docker-vxlan-plugin"]
