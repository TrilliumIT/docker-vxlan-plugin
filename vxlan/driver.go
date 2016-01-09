package vxlan

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	dknet "github.com/docker/go-plugins-helpers/network"
	"github.com/samalba/dockerclient"
	"github.com/socketplane/libovsdb"
	"github.com/vishvananda/netlink"
)

const (
	mtuOption        = "mtu"
	bridgeNameOption = "bridgeName"
	srcAddrOption    = "srcAddr"
	groupOption      = "group"
	ttlOption        = "ttl"
	tosOption        = "tos"

	bindInterfaceOption = "net.gopher.ovs.bridge.bind_interface"
	defaultMTU          = 1500
)

type Driver struct {
	dknet.Driver
	networks map[string]*NetworkState
}

// NetworkState is filled in at network creation time
// it contains state that we wish to keep for each network
type NetworkState struct {
	BridgeName string
	VXLanName  string
}

func (d *Driver) CreateNetwork(r *dknet.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)

	//if r.Options == nil {
	//	return "", fmt.Errorf("No options provided")
	//}

	//vxlanID := "42"
	//if r.Options["vxlanID"] != nil {
	//	vxlanID = r.Options["vxlanID"]
	//}
}
