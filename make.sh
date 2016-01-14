docker build -f ./Dockerbuild -t docker-vxlan-plugin-build . && docker run docker-vxlan-plugin-build cat /go/bin/docker-vxlan-plugin > docker-vxlan-plugin
chmod +x docker-vxlan-plugin
