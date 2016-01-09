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
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return err
	}

	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{Name: vxlanName},
		VxlanId:   vxlanID,
	}
	err = netlink.LinkAdd(vxlan)
	if err != nil {
		return err
	}

	ns := &NetworkState{
		VXLan:  vxlan,
		Bridge: bridge,
	}
	d.networks[name] = ns

	err = netlink.LinkSetMaster(vxlan, bridge)
	if err != nil {
		return err
	}

	return nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	name := r.NetworkID

	err := netlink.LinkDel(d.networks[name].VXLan)
	if err != nil {
		return err
	}
	err = netlink.LinkDel(d.networks[name].Bridge)
	if err != nil {
		return err
	}

	return nil
}
