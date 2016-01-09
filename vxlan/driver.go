package vxlan

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/network"
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
	network.Driver
	networks map[string]*NetworkState
}

// NetworkState is filled in at network creation time
// it contains state that we wish to keep for each network
type NetworkState struct {
	BridgeName	string
	VXLanName	string
	VXLanID		int
}

func NewDriver() (*Driver, error) {
	d := &Driver{
		networks: make(map[string]*NetworkState),
	}
	return d, nil
}

// NewHandler initializes the request handler with a driver implementation.
func NewHandler(driver Driver) *Handler {
	h := &Handler{driver, sdk.NewHandler(manifest)}
	h.initMux()
	return h
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)

	name := r.NetworkID

	//if r.Options == nil {
	//	return "", fmt.Errorf("No options provided")
	//}

	vxlanName := "vxlan42"
	vxlanID := 42
	//if r.Options["vxlanID"] != nil {
	//	vxlanID = r.Options["vxlanID"]
	//}

	BridgeName := "br_vxlan42"


	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: ns['BridgeName'] },
	}
	netlink.LinkAdd(bridge)

	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{Name: ns['VXlanName']},
		VxlanId:   ns['VXLanID'],
	}
	netlink.LinkAdd(vxlan)

	ns := &NetworkState {
		VXLan:		vxlan,
		Bridge:		bridge,
	}
	d.networks[r.NetworkID] = ns

	netlink.LinkSetMaster(vxlan, bridge)

}
