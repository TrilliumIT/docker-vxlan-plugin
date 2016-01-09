package vxlan

import (
	//"fmt"
	//"strings"
	//"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/network"
	//"github.com/samalba/dockerclient"
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
	network.Driver
	networks map[string]*NetworkState
}

// NetworkState is filled in at network creation time
// it contains state that we wish to keep for each network
type NetworkState struct {
	Bridge *netlink.Bridge
	VXLan  *netlink.Vxlan
}

func NewDriver() (*Driver, error) {
	d := &Driver{
		networks: make(map[string]*NetworkState),
	}
	return d, nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)

	//if r.Options == nil {
	//	return "", fmt.Errorf("No options provided")
	//}

	vxlanName := "vxlan42"
	vxlanID := 42
	//if r.Options["vxlanID"] != nil {
	//	vxlanID = r.Options["vxlanID"]
	//}

	bridgeName := "br_vxlan42"

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: bridgeName},
	}
	netlink.LinkAdd(bridge)

	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{Name: vxlanName},
		VxlanId:   vxlanID,
	}
	netlink.LinkAdd(vxlan)

	ns := &NetworkState{
		VXLan:  vxlan,
		Bridge: bridge,
	}
	d.networks[r.NetworkID] = ns

	netlink.LinkSetMaster(vxlan, bridge)

	return nil
}
