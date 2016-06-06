docker-vxlan-plugin is a vxlan plugin for docker designed for on premise deployments where users have full control over their network. It is purposefully built so that users can manage routing as part of their larger infrastructure, and avoid NAT and multi-homed containers. This plugin alone handles communication between containers on a single vxlan, even (directly) between hosts. It requires an external routing system to route between vxlans. Please see our work on [docker-drouter](https://github.com/TrilliumIT/docker-drouter/) for a distributed routing system that enables short-cut routing between vxlans.

## Use Cases

The out of box networking options for docker are built around a use case common to deployment on managed virtual servers where each host has a single public IP address. They are not well suited to bare metal datacenter deployments where routing can be controlled and multiple layers of nat is undesirable. This plugin does not nat your containers and assumes that you're running it in an environment where you know how to distribute routes to your vxlan network. You must set the gateway to the IP address of another router that already exists on your vxlan network.

## Quick start

Follow the [Tutorial](tutorial.md) for how to get up and running quickly with docker-machine.

## How it works

When a container joins a network created with the vxlan driver if doesn't already exist a [vxlan interface](https://www.kernel.org/doc/Documentation/networking/vxlan.txt) is created. [MacVlan](http://www.pocketnix.org/posts/Linux%20Networking:%20MAC%20VLANs%20and%20Virtual%20Ethernets) interfaces are created for each container and attached to the vxlan interface.

## Running the plugin

### Run from docker

```sh
docker run -v /run/docker/plugins/:/run/docker/plugins -v /var/run/docker.sock:/var/run/docker.sock --privileged  --net=host TrilliumIT/docker-vxlan-plugin
```

The plugin must be run in privileged mode with host networking to be able to add network links to the system.

### Run outside docker

Download the latest release binary and execute it.

## Options

### Daemon Options

#### -d

debug output

#### -scope

scope of the plugin. Can be either `local` or `global`. Default is `local`. If `-scope=global` is specified the network options will be published to the docker cluster key-value store and containers can be brought up on the network on any host in the cluster after the network has been created. The global scope will also allow the default global IPAM driver to be used which will coordinate IP address allocation between all hosts in a docker cluster. Note that the gateway address specified during network creation will not be assigned to the host, but it will still be passed to containers as their default route. This applies in both `global` and `local` mode.

#### -vtepdev

The device to use as the VTep endpoint. If this is specified as a daemon option it takes presidence over a VtepDev specified as a `network create` option.

### Network create options

The following options can be passed to `docker network create` as `-o option=value`. Please consult the man page for ip link and see the vxlan section for more details on some of these options.

#### vxlanName

Name of the vxlan interface

#### vxlanMTU

MTU of the vxlan interface

#### vxlanHardwareAddr

MAC Address of the vxlan interface

#### vxlanTxQLen

Transaction Queue Length of the vxlan interface

#### VxlanId

specifies the VXLAN Network Identifer (or VXLAN Segment Identifier) to use.

#### VtepDev

specifies the physical device to use for tunnel endpoint communication.

#### SrcAddr

specifies the source IP address to use in outgoing packets.

#### Group

specifies the multicast IP address to join.

#### TTL

specifies the TTL value to use in outgoing packets.

#### TOS

specifies the TOS value to use in outgoing packets.

#### Learning

specifies if unknown source link layer addresses and IP addresses are entered into the VXLAN device forwarding database.

#### Proxy

specifies ARP proxy is turned on.

#### RSC

specifies if route short circuit is turned on.

#### L2miss

specifies if netlink LLADDR miss notifications are generated.

#### L3miss

specifies if netlink IP ADDR miss notifications are generated.

#### NoAge

Do not age FDB entries.

#### GBP

enables the Group Policy extension (VXLAN-GBP).

#### Age

specifies the lifetime in seconds of FDB entries learnt by the kernel.

#### Limit

specifies the maximum number of FDB entries.

#### PortLow

specifies the minimum UDP source port

#### PortHigh

specifies the maximum UDP source port
