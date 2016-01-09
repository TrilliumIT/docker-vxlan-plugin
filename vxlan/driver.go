package vxlan

import (
	//"fmt"
	//"strings"
	//"time"
	"net"

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
			vxlanName = r.Options["vxlanName"].(string)
		}
		if r.Options["bridgeName"] != nil {
			bridgeName = r.Options["bridgeName"].(string)
		}
	}
	spew.dump(r.Options["vxlanName"])

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: bridgeName},
	}
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{Name: vxlanName},
	}

	if r.Options != nil {
		if r.Options["VxlanId"] != nil {
			vxlan.VxlanId = r.Options["VxlanId"].(int)
		}
		if r.Options["VtepDev"] != nil {
			vtepDev, err := netlink.LinkByName(r.Options["VtepDev"].(string))
			if err != nil {
				return err
			}
			vxlan.VtepDevIndex = vtepDev.Attrs().Index
		}
		if r.Options["SrcAddr"] != nil {
			vxlan.SrcAddr = net.ParseIP(r.Options["SrcAddr"].(string))
		}
		if r.Options["Group"] != nil {
			vxlan.Group = net.ParseIP(r.Options["Group"].(string))
		}
		if r.Options["TTL"] != nil {
			vxlan.TTL = r.Options["TTL"].(int)
		}
		if r.Options["TOS"] != nil {
			vxlan.TOS = r.Options["TOS"].(int)
		}
		if r.Options["Learning"] != nil {
			vxlan.Learning = r.Options["Learning"].(bool)
		}
		if r.Options["Proxy"] != nil {
			vxlan.Proxy = r.Options["Proxy"].(bool)
		}
		if r.Options["RSC"] != nil {
			vxlan.RSC = r.Options["RSC"].(bool)
		}
		if r.Options["L2miss"] != nil {
			vxlan.L2miss = r.Options["L2miss"].(bool)
		}
		if r.Options["L3miss"] != nil {
			vxlan.L3miss = r.Options["L3miss"].(bool)
		}
		if r.Options["NoAge"] != nil {
			vxlan.NoAge = r.Options["NoAge"].(bool)
		}
		if r.Options["GBP"] != nil {
			vxlan.GBP = r.Options["BGP"].(bool)
		}
		if r.Options["Age"] != nil {
			vxlan.Age = r.Options["Age"].(int)
		}
		if r.Options["Limit"] != nil {
			vxlan.Limit = r.Options["Limit"].(int)
		}
		if r.Options["Port"] != nil {
			vxlan.Port = r.Options["Port"].(int)
		}
		if r.Options["PortLow"] != nil {
			vxlan.PortLow = r.Options["PortLow"].(int)
		}
		if r.Options["PortHigh"] != nil {
			vxlan.PortHigh = r.Options["PortHigh"].(int)
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
