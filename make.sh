docker run . cat /go/bin/docker-vxlan-plugin >> docker-vxlan-plugin
docker build -t docker-vxlan-plugin . && docker run docker-vxlan-plugin cat /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
