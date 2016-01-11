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
	//"github.com/davecgh/go-spew/spew"
	"github.com/vishvananda/netlink"
)

type Driver struct {
	network.Driver
	networks map[string]*NetworkState
}

// NetworkState is filled in at network creation time
// it contains state that we wish to keep for each network
type NetworkState struct {
	Bridge   *netlink.Bridge
	VXLan    *netlink.Vxlan
	Gateway  string
	IPv4Data []*network.IPAMData
	IPv6Data []*network.IPAMData
}

func NewDriver() (*Driver, error) {
	d := &Driver{
		networks: make(map[string]*NetworkState),
	}
	return d, nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)

	netID := r.NetworkID
	var err error

	bridgeName := "br_" + netID[:12]
	vxlanName := "vx_" + netID[:12]

	// get interface names from options first
	for k, v := range r.Options {
		if k == "com.docker.network.generic" {
			if genericOpts, ok := v.(map[string]interface{}); ok {
				for key, val := range genericOpts {
					if key == "vxlanName" {
						vxlanName = val.(string)
					}
					if key == "bridgeName" {
						bridgeName = val.(string)
					}
				}
			}
		}
	}

	// create links
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
		},
	}
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: vxlanName,
		},
	}

	// Parse interface options
	for k, v := range r.Options {
		if k == "com.docker.network.generic" {
			if genericOpts, ok := v.(map[string]interface{}); ok {
				for key, val := range genericOpts {
					log.Debugf("Libnetwork Opts Sent: [ %s ] Value: [ %s ]", key, val)
					if key == "vxlanMTU" {
						vxlan.LinkAttrs.MTU, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "bridgeMTU" {
						bridge.LinkAttrs.MTU, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "vxlanHardwareAddr" {
						vxlan.LinkAttrs.HardwareAddr, err = net.ParseMAC(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "bridgeHardwareAddr" {
						bridge.LinkAttrs.HardwareAddr, err = net.ParseMAC(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "vxlanTxQLen" {
						vxlan.LinkAttrs.TxQLen, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "bridgeTxQLen" {
						bridge.LinkAttrs.TxQLen, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "VxlanId" {
						vxlan.VxlanId, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "VtepDev" {
						vtepDev, err := netlink.LinkByName(val.(string))
						if err != nil {
							return err
						}
						vxlan.VtepDevIndex = vtepDev.Attrs().Index
					}
					if key == "SrcAddr" {
						vxlan.SrcAddr = net.ParseIP(val.(string))
					}
					if key == "Group" {
						vxlan.Group = net.ParseIP(val.(string))
					}
					if key == "TTL" {
						vxlan.TTL, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "TOS" {
						vxlan.TOS, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "Learning" {
						vxlan.Learning, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "Proxy" {
						vxlan.Proxy, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "RSC" {
						vxlan.RSC, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "L2miss" {
						vxlan.L2miss, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "L3miss" {
						vxlan.L3miss, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "NoAge" {
						vxlan.NoAge, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "GBP" {
						vxlan.GBP, err = strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "Age" {
						vxlan.Age, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "Limit" {
						vxlan.Limit, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "Port" {
						vxlan.Port, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "PortLow" {
						vxlan.PortLow, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
					if key == "PortHigh" {
						vxlan.PortHigh, err = strconv.Atoi(val.(string))
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	// Done parsing options

	// delete links if they already exist, don't worry about errors
	netlink.LinkDel(bridge)
	netlink.LinkDel(vxlan)

	// add links
	err = netlink.LinkAdd(bridge)
	if err != nil {
		return err
	}
	err = netlink.LinkAdd(vxlan)
	if err != nil {
		return err
	}

	// add vxlan to bridge
	err = netlink.LinkSetMaster(vxlan, bridge)
	if err != nil {
		return err
	}

	// bring interfaces up
	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return err
	}
	err = netlink.LinkSetUp(vxlan)
	if err != nil {
		return err
	}

	// store interfaces to be used later
	ns := &NetworkState{
		VXLan:    vxlan,
		Bridge:   bridge,
		IPv4Data: r.IPv4Data,
		IPv6Data: r.IPv6Data,
	}

	// Add IPs to interfaces
	// Process IPv6 first. If both are inclued, IPv4 gateway will be the one that
	// remains, because JoinResponse can only include one Gateway
	for i := range r.IPv6Data {
		gatewayIP, err := netlink.ParseAddr(r.IPv6Data[i].Gateway)
		if err != nil {
			return err
		}
		ns.Gateway = gatewayIP.IP.String()
		netlink.AddrAdd(bridge, gatewayIP)
	}
	for i := range r.IPv4Data {
		gatewayIP, err := netlink.ParseAddr(r.IPv4Data[i].Gateway)
		if err != nil {
			return err
		}
		ns.Gateway = gatewayIP.IP.String()
		netlink.AddrAdd(bridge, gatewayIP)
	}

	d.networks[netID] = ns

	return nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	netID := r.NetworkID

	err := netlink.LinkDel(d.networks[netID].VXLan)
	if err != nil {
		return err
	}
	err = netlink.LinkDel(d.networks[netID].Bridge)
	if err != nil {
		return err
	}

	return nil
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) error {
	log.Debugf("Create endpoint request: %+v", r)
	return nil
}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
	log.Debugf("Delete endpoint request: %+v", r)
	return nil
}

func (d *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
	res := &network.InfoResponse{
		Value: make(map[string]string),
	}
	return res, nil
}

func (d *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
	// create and attach local name to the bridge
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth_" + r.EndpointID[:5]},
		PeerName:  "ethc" + r.EndpointID[:5],
	}
	if err := netlink.LinkAdd(veth); err != nil {
		log.Errorf("failed to create the veth pair named: [ %v ] error: [ %s ] ", veth, err)
		return nil, err
	}

	// bring up the veth pair
	err := netlink.LinkSetUp(veth)
	if err != nil {
		log.Warnf("Error enabling  Veth local iface: [ %v ]", veth)
		return nil, err
	}

	bridge := d.networks[r.NetworkID].Bridge
	// add veth to bridge
	err = netlink.LinkSetMaster(veth, bridge)
	if err != nil {
		return nil, err
	}

	// SrcName gets renamed to DstPrefix + ID on the container iface
	res := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   veth.PeerName,
			DstPrefix: "eth",
		},
		Gateway: d.networks[r.NetworkID].Gateway,
	}
	log.Debugf("Join endpoint %s:%s to %s", r.NetworkID, r.EndpointID, r.SandboxKey)
	return res, nil
}

func (d *Driver) Leave(r *network.LeaveRequest) error {
	log.Debugf("Leave request: %+v", r)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth_" + r.EndpointID[:5]},
		PeerName:  "ethc" + r.EndpointID[:5],
	}

	// bring down the veth pair
	err := netlink.LinkSetDown(veth)
	if err != nil {
		log.Warnf("Error bring down Veth local iface: [ %v ]", veth)
		return err
	}

	// remove veth from bridge
	err = netlink.LinkSetNoMaster(veth)
	if err != nil {
		log.Warnf("Error removing veth from bridge")
		return err
	}

	// delete the veth interface
	err = netlink.LinkDel(veth)
	if err != nil {
		log.Warnf("Error removing veth interface")
		return err
	}

	return nil
}
