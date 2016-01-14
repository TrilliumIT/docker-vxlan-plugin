This guide assumes you're working on a linux host. You will first build a few docker hosts in a cluster using docker-machine. Then the host itself will get an interface on vxlan network. At the end communication to the docker containers via their container IP will be possible without NAT from your host outside the docker swarm will be possible.

1. Setup docker-machine

Install docker, docker-machine and docker-compose and virtualbox. Start by creating a docker-machine which will be used to generate your token to create a swarm cluster.

```
$ docker-machine create -d virtualbox default
$ eval "$(docker-machine env default)"
$ TOKEN=$(docker run swarm create 2> /dev/null)
```

This will store a unique swarm token as $TOKEN in your shell.

2. Create your swarm manager. Note the `iptables=false` and `ip-masq=false` these tell Docker to not NAT our containers and not mess with the iptables rules. This guide assumes you know how and want to manage your NAT and firewall yourself.

```
$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-master \
    --swarm-discovery token://$TOKEN \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-master
```

3.  Create a couple of swarm agent hosts:

```
$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery token://$TOKEN \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-agent-00

$ docker-machine create \
    -d virtualbox \
    --swarm \
    --swarm-discovery token://$TOKEN \
    --engine-opt iptables=false \
    --engine-opt ip-masq=false \
    swarm-agent-01
```

4. Set your shell to manage the swarm:

```
$ eval $(docker-machine env --swarm swarm-master)
```

5. Now your docker swarm is up and running. Time to run the vxlan driver.
