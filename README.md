docker-vxlan-plugin is a vxlan plugin for docker that enables plumbing docker containers into an existing vxlan network.

## Run from docker ##
docker run -v /run/docker/plugins/:/run/docker/plugins -v /var/run/docker.sock:/var/run/docker.sock clinta/docker-vxlan-plugin
