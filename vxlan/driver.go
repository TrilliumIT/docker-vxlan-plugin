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

	name := r.NetworkID

	//if r.Options == nil {
	//	return "", fmt.Errorf("No options provided")
	//}

	vxlanName := "vx_42" // + name
	vxlanID := 42
	//if r.Options["vxlanID"] != nil {
	//	vxlanID = r.Options["vxlanID"]
	//}

	bridgeName := "br_42" // + name

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
	d.networks[name] = ns

	netlink.LinkSetMaster(vxlan, bridge)

	return nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	name := r.NetworkID

	netlink.LinkDel(d.networks[name].VXLan)
	netlink.LinkDel(d.networks[name].Bridge)

	return nil
}
