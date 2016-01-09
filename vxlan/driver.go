package vxlan

import (
	//"fmt"
	//"strings"
	//"time"
	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/network"
	//"github.com/samalba/dockerclient"
	"github.com/davecgh/go-spew/spew"
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
	spew.Dump(r)

	name := r.NetworkID[0:12]

	vxlanName := "vx_" + name
	bridgeName := "br_" + name

	if r.Options != nil {
		if r.Options["vxlanName"] != nil {
			vxlanName = r.Options["vxlanName"]
		}
		if r.Options["bridgeName"] != nil {
			bridgeName = r.Options["bridgeName"]
		}
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: bridgeName},
	}
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{Name: vxlanName},
	}

	if r.Options != nil {
		if r.Options["VxlanID"] != nil {
			vxlan.VxlanId = strconf.ParseUInt(r.Options["VxlanID"])
		}
		if r.Options["VtepDev"] != nil {
			vtepDev = netlink.LinkByName(r.Options["VtepDev"]
			vxlan.VtepDevIndex = vtepDev.Attrs().Index
		}
		if r.Options["SrcAddr"] != nil {
			vxlan.SrcAddr = net.ParseIP(r.Options["SrcAddr"])
		}
		if r.Options["Group"] != nil {
			vxlan.Group = net.ParseIP(r.Options["Group"])
		}
		if r.Options["TTL"] != nil {
			vxlan.TTL = strconf.ParseUInt(r.Options["TTL"])
		}
		if r.Options["TOS"] != nil {
			vxlan.TOS = strconf.ParseUInt(r.Options["TOS"])
		}
		if r.Options["Learning"] != nil {
			vxlan.Learning = strconf.ParseBool(r.Options["Learning"])
		}
		if r.Options["Proxy"] != nil {
			vxlan.Proxy = strconf.ParseBool(r.Options["Proxy"])
		}
		if r.Options["RSC"] != nil {
			vxlan.RSC = strconf.ParseBool(r.Options["RSC"])
		}
		if r.Options["L2miss"] != nil {
			vxlan.L2miss = strconf.ParseBool(r.Options["L2miss"])
		}
		if r.Options["L3miss"] != nil {
			vxlan.L3miss = strconf.ParseBool(r.Options["L2miss"])
		}
		if r.Options["NoAge"] != nil {
			vxlan.NoAge = strconf.ParseBool(r.Options["NoAge"])
		}
		if r.Options["BGP"] != nil {
			vxlan.BGP = strconf.ParseBool(r.Options["BGP"])
		}
		if r.Options["Age"] != nil {
			vxlan.Age = strconf.ParseUInt(r.Options["Age"])
		}
		if r.Options["Limit"] != nil {
			vxlan.Limit = strconf.ParseUInt(r.Options["Limit"])
		}
		if r.Options["Port"] != nil {
			vxlan.Port = strconf.ParseUInt(r.Options["Port"])
		}
		if r.Options["PortLow"] != nil {
			vxlan.PortLow = strconf.ParseUInt(r.Options["PortLow"])
		}
		if r.Options["PortHigh"] != nil {
			vxlan.PortHigh = strconf.ParseUInt(r.Options["PortHigh"])
		}
	}

	err := netlink.LinkAdd(bridge)
	if err != nil {
		return err
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
	name := r.NetworkID[0:12]

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
