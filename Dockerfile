FROM debian:jessie
ENV VER=v0.5.1

MAINTAINER Clint Armstrong <clint@clintarmstrong.net>

RUN apt-get -qq update && \
    apt-get -yqq install arptables kmod && \
    apt-get -qq clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ADD https://github.com/clinta/docker-vxlan-plugin/releases/download/${VER}/docker-vxlan-plugin /docker-vxlan-plugin
RUN chmod +x /docker-vxlan-plugin

ENTRYPOINT ["/docker-vxlan-plugin"]
