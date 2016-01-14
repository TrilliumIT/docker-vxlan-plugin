This guide assumes you're working on a linux host. You will first build a few docker hosts in a cluster using docker-machine. Then the host itself will get an interface on vxlan network. At the end communication to the docker containers via their container IP will be possible without NAT from your host outside the docker swarm will be possible.

1. Setup docker-machine

Install docker, docker-machine and docker-compose and virtualbox. Start by creating a docker-machine which will hold a kv store used for swarm and overlay network configuration.

```bash
$ docker-machine create -d virtualbox mh-keystore
$ docker $(docker-machine config mh-keystore) run -d \
    -p "8500:8500" \
    -h "consul" \
    progrium/consul -server -bootstrap
```

2. Create your swarm manager. Note the `iptables=false` and `ip-masq=false` these tell Docker to not NAT our containers and not mess with the iptables rules. This guide assumes you know how and want to manage your NAT and firewall yourself.

```
$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-master \
    --swarm-discovery "consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-store=consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-master
```

3.  Create a couple of swarm agent hosts:

```
$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery "consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-store=consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-agent-00

$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery "consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-store=consul://$(docker-machine ip mh-keystore):8500" \
    --engine-opt="cluster-advertise=eth1:2376" \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-agent-01
```

4. Set your shell to manage the swarm:

```
$ eval $(docker-machine env --swarm swarm-master)
```

5. Now your docker swarm is up and running. Time to run the vxlan driver. You need to run this on every node in the swarm, so we'll use [constraints](https://docs.docker.com/swarm/scheduler/filter/) to do it.

```
$ docker run -d --restart=unless-stopped -e constraint:node==swarm-master --net=host --privileged -v /run/docker/plugins/:/run/docker/plugins -v /var/run/docker.sock:/var/run/docker.sock clinta/docker-vxlan-plugin:latest -scope=global
$ docker run -d --restart=unless-stopped -e constraint:node==swarm-agent-00 --net=host --privileged -v /run/docker/plugins/:/run/docker/plugins -v /var/run/docker.sock:/var/run/docker.sock clinta/docker-vxlan-plugin:latest -scope=global
$ docker run -d --restart=unless-stopped -e constraint:node==swarm-agent-01 --net=host --privileged -v /run/docker/plugins/:/run/docker/plugins -v /var/run/docker.sock:/var/run/docker.sock clinta/docker-vxlan-plugin:latest -scope=global
```

6. Now you can create a docker vxlan network for containers to use to communicate between hosts.

```
$ docker network create -d vxlan -o Group=239.1.1.1 -o VxlanId=42 -o VtepDev=eth1 --subnet=10.42.0.0/24 --gateway=10.42.0.254 vxlan42
```

7. Start some containers on the vxlan network. We'll use constraints again just to prove that containers can communicate between hosts, though you would likely not pin containers to hosts in a real deployment.

```
$ docker run -d --name=swarm-master-nginx --net=vxlan42 -e constraint:node==swarm-master nginx
$ docker run -d --name=swarm-agent-00-nginx --net=vxlan42 -e constraint:node==swarm-agent-00 nginx
$ docker run -d --name=swarm-agent-01-nginx --net=vxlan42 -e constraint:node==swarm-agent-01 nginx
```

You can checkout the IP addresses of these containers with `docker exec`

```
$ docker exec swarm-master-nginx ip addr
$ docker exec swarm-agent-00-nginx ip addr
$ docker exec swarm-agent-01-nginx ip addr
```

In my cases the IPs are 10.42.0.1, 10.42.0.2 and 10.42.0.3. To verify connectivity I'll ping between them.

```
$ docker exec -it swarm-master-nginx ping -c 4 10.42.0.2
$ docker exec -it swarm-master-nginx ping -c 4 10.42.0.3
$ docker exec -it swarm-agent-00-nginx ping -c 4 10.42.0.1
$ docker exec -it swarm-agent-00-nginx ping -c 4 10.42.0.3
$ docker exec -it swarm-agent-01-nginx ping -c 4 10.42.0.2
$ docker exec -it swarm-agent-01-nginx ping -c 4 10.42.0.3
```

8. At this point, containers can communicate amongst themselves, but your host cannot communicate with them. Try and ping any of the 10.42.0.1-3 addresses and it will fail. To get communication to your hosts, you simply have to add a vxlan interface. This is the same procedure you might use to add the vxlan interface to a router in a production environment.

```
# ip link add vxlan42 type vxlan id 42 dev vboxnet0 group 239.1.1.1
# ip link set up vxlan42
# ip addr add 10.42.0.254/24 dev vxlan42
```

Now you have connectivity between your host and containers with zero NAT. Try pinging, or using a web browser to pull up the pages on 10.42.0.[1-3].
