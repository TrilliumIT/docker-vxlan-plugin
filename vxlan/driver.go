package vxlan

import (
	gonet "net"
	"strconv"
	"errors"
	"strings"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/samalba/dockerclient"
	"github.com/vishvananda/netlink"
)

type Driver struct {
	network.Driver
	scope	          string
	vtepdev           string
	allow_empty       bool
	global_gateway    bool
	block_gateway_arp bool
	networks          map[string]*NetworkState
	docker	          *dockerclient.DockerClient
}

// NetworkState is filled in at network creation time
// it contains state that we wish to keep for each network
type NetworkState struct {
	Bridge	 *netlink.Bridge
	VXLan	 *netlink.Vxlan
	Gateway  string
	IPv4Data []*network.IPAMData
	IPv6Data []*network.IPAMData
}

func NewDriver(scope string, vtepdev string, allow_empty bool, global_gateway bool, block_gateway_arp bool) (*Driver, error) {
	docker, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		return nil, err
	}
	d := &Driver{
		scope: scope,
		vtepdev: vtepdev,
		allow_empty: allow_empty,
		global_gateway: global_gateway,
		networks: make(map[string]*NetworkState),
		docker: docker,
	}
	if d.allow_empty {
		nets, err := d.docker.ListNetworks("")
		log.Debugf("Nets: %+v", nets)
		if err != nil {
			return d, err
		}
		for i := range nets {
			if nets[i].Driver == "vxlan" {
				log.Debugf("Net[i]: %+v", nets[i])
				_, err := d.getLinks(nets[i].ID)
				if err != nil {
					return d, err
				}
			}
		}
	}
	return d, nil
}

func (d *Driver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Debugf("Get Capabilities request")
	res := &network.CapabilitiesResponse{
		Scope: d.scope,
	}
	log.Debugf("Responding with %+v", res)
	return res, nil
}

type intNames struct {
	VxlanName  string
	BridgeName string
}

func getIntNames(netID string, docker *dockerclient.DockerClient) (*intNames, error) {
	net, err := docker.InspectNetwork(netID)
	if err != nil {
		return nil, err
	}

	names := &intNames{}

	if net.Driver != "vxlan" {
		log.Errorf("Network %v is not a vxlan network", netID)
		return nil, errors.New("Not a vxlan network")
	}

	names.BridgeName = "br_" + netID[:12]
	names.VxlanName = "vx_" + netID[:12]

	// get interface names from options first
	for k, v := range net.Options {
		if k == "vxlanName" {
			names.VxlanName = v
		}
		if k == "bridgeName" {
			names.BridgeName = v
		}
	}

	return names, nil
}

func getGateway(netID string, docker dockerclient.DockerClient) (string, error) {
	net, err := docker.InspectNetwork(netID)
	if err != nil {
		return "", err
	}

	for i := range net.IPAM.Config {
		if net.IPAM.Config[i].Gateway != "" {
			return net.IPAM.Config[i].Gateway, nil
		}
	}
	return "", nil
}

type intLinks struct {
	Vxlan  *netlink.Vxlan
	Bridge *netlink.Bridge
}

// this function gets netlink devices or creates them if they don't exist
func (d *Driver) getLinks(netID string) (*intLinks, error) {
	docker := d.docker
	net, err := docker.InspectNetwork(netID)
	if err != nil {
		return nil, err
	}

	if net.Driver != "vxlan" {
		log.Errorf("Network %v is not a vxlan network", netID)
		return nil, errors.New("Not a vxlan network")
	}

	names, err := getIntNames(netID, docker)
	if err != nil {
		return nil, err
	}

	// get or create links
	var bridge *netlink.Bridge
	bridgelink, err := netlink.LinkByName(names.BridgeName)
	if err == nil {
		bridge = &netlink.Bridge{
			LinkAttrs: *bridgelink.Attrs(),
		}
	} else {
		bridge, err = d.createBridge(names.BridgeName, net)
		if err != nil {
			return nil, err
		}
	}
	var vxlan *netlink.Vxlan
	vxlanlink, err := netlink.LinkByName(names.VxlanName)
	if err == nil {
		vxlan = &netlink.Vxlan{
			LinkAttrs: *vxlanlink.Attrs(),
		}
	} else {
		vxlan, err = d.createVxLan(names.VxlanName, net)
		if err != nil {
			return nil, err
		}
	}

	// add vxlan to bridge
	if vxlan.LinkAttrs.MasterIndex == 0 {
		err = netlink.LinkSetMaster(vxlan, bridge)
		if err != nil {
			return nil, err
		}
	}

	links := &intLinks{
		Vxlan: vxlan,
		Bridge: bridge,
	}

	return links, nil
}

func (d *Driver) createBridge(bridgeName string, net *dockerclient.NetworkResource) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
		},
	}
	// Parse interface options
	for k, v := range net.Options {
		if k == "bridgeMTU" {
			mtu, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			bridge.LinkAttrs.MTU = mtu
		}
		if k == "bridgeHardwareAddr" {
			hardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				return nil, err
			}
			bridge.LinkAttrs.HardwareAddr = hardwareAddr
		}
		if k == "bridgeTxQLen" {
			txQLen, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			bridge.LinkAttrs.TxQLen = txQLen
		}
	}

	err := netlink.LinkAdd(bridge)
	if err != nil {
		return nil, err
	}

	// Parse interface options
	for k, v := range net.Options {
		if k == "bridgeHardwareAddr" {
			hardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				return nil, err
			}
			err = netlink.LinkSetHardwareAddr(bridge, hardwareAddr)
			if err != nil {
				return nil, err
			}
		}
		if k == "bridgeMTU" {
			mtu, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			err = netlink.LinkSetMTU(bridge, mtu)
			if err != nil {
				return nil, err
			}
		}
	}

	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return nil, err
	}

	if d.scope == "local" || d.global_gateway {
		for i := range net.IPAM.Config {
			mask := strings.Split(net.IPAM.Config[i].Subnet, "/")[1]
			gatewayIP, err := netlink.ParseAddr(net.IPAM.Config[i].Gateway + "/" + mask)
			if err != nil {
				return nil, err
			}
			netlink.AddrAdd(bridge, gatewayIP)
		}
	}

	return bridge, nil
}

func (d *Driver) createVxLan(vxlanName string, net *dockerclient.NetworkResource) (*netlink.Vxlan, error) {
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: vxlanName,
		},
	}

	// Parse interface options
	for k, v := range net.Options {
		if k == "vxlanMTU" {
			MTU, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.LinkAttrs.MTU = MTU
		}
		if k == "vxlanHardwareAddr" {
			HardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				return nil, err
			}
			vxlan.LinkAttrs.HardwareAddr = HardwareAddr
		}
		if k == "vxlanTxQLen" {
			TxQLen, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.LinkAttrs.TxQLen = TxQLen
		}
		if k == "VxlanId" {
			log.Debugf("VxlanID: %+v", v)
			VxlanId, err := strconv.ParseInt(v, 0, 32)
			if err != nil {
				return nil, err
			}
			log.Debugf("VxlanID: %+v", VxlanId)
			log.Debugf("int(VxlanID): %+v", int(VxlanId))
			vxlan.VxlanId = int(VxlanId)
		}
		if k == "VtepDev" {
			vtepDev, err := netlink.LinkByName(v)
			if err != nil {
				return nil, err
			}
			vxlan.VtepDevIndex = vtepDev.Attrs().Index
		}
		if k == "SrcAddr" {
			vxlan.SrcAddr = gonet.ParseIP(v)
		}
		if k == "Group" {
			vxlan.Group = gonet.ParseIP(v)
		}
		if k == "TTL" {
			TTL, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.TTL = TTL
		}
		if k == "TOS" {
			TOS, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.TOS = TOS
		}
		if k == "Learning" {
			Learning, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.Learning = Learning
		}
		if k == "Proxy" {
			Proxy, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.Proxy = Proxy
		}
		if k == "RSC" {
			RSC, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.RSC = RSC
		}
		if k == "L2miss" {
			L2miss, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.L2miss = L2miss
		}
		if k == "L3miss" {
			L3miss, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.L3miss = L3miss
		}
		if k == "NoAge" {
			NoAge, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.NoAge = NoAge
		}
		if k == "GBP" {
			GBP, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vxlan.GBP = GBP
		}
		if k == "Age" {
			Age, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.Age = Age
		}
		if k == "Limit" {
			Limit, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.Limit = Limit
		}
		if k == "Port" {
			Port, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.Port = Port
		}
		if k == "PortLow" {
			PortLow, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.PortLow = PortLow
		}
		if k == "PortHigh" {
			PortHigh, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			vxlan.PortHigh = PortHigh
		}
	}

	if d.vtepdev != "" {
		vtepDev, err := netlink.LinkByName(d.vtepdev)
		if err != nil {
			return nil, err
		}
		vxlan.VtepDevIndex = vtepDev.Attrs().Index
	}

	err := netlink.LinkAdd(vxlan)
	if err != nil {
		return nil, err
	}

	// Parse interface options
	for k, v := range net.Options {
		if k == "vxlanHardwareAddr" {
			hardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				return nil, err
			}
			err = netlink.LinkSetHardwareAddr(vxlan, hardwareAddr)
			if err != nil {
				return nil, err
			}
		}
		if k == "vxlanMTU" {
			mtu, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			err = netlink.LinkSetMTU(vxlan, mtu)
			if err != nil {
				return nil, err
			}
		}
	}

	if d.block_gateway_arp {
		gatewayIP := ""
		for i := range net.IPAM.Config {
			if net.IPAM.Config[i].Gateway != "" {
				gatewayIP = net.IPAM.Config[i].Gateway
			}
		}

		if gatewayIP != "" {

			cmd := exec.Command(	"arptables",
						"--append", "FORWARD",
						"--out-interface", vxlanName,
						"--destination", gatewayIP,
						"--opcode", "1",
						"--jump", "DROP" )
			err = cmd.Run()
			if err != nil {
				return nil, err
			}

			cmd = exec.Command(	"arptables",
						"--append", "FORWARD",
						"--in-interface", vxlanName,
						"--source", gatewayIP,
						"--opcode", "2",
						"--jump", "DROP" )
			err = cmd.Run()
			if err != nil {
				return nil, err
			}
		}
	}

	// bring interfaces up
	err = netlink.LinkSetUp(vxlan)
	if err != nil {
		return nil, err
	}

	return vxlan, nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)

	// return nil and lazy create the network when a container joins it
	// Active creation when allow_empty is enabled will be handled by watching libkv
	return nil
}

func (d *Driver) deleteNics(netID string) error {
	names, err := getIntNames(netID, d.docker)
	if err != nil {
		return err
	}

	vxlan, err := netlink.LinkByName(names.VxlanName)
	if err == nil {
		err := netlink.LinkDel(vxlan)
		if err != nil {
			return err
		}
		log.Debugf("Deleting interface %+v", names.VxlanName)
	}
	bridge, err := netlink.LinkByName(names.BridgeName)
	if err == nil {
		err := netlink.LinkDel(bridge)
		if err != nil {
			return err
		}
		log.Debugf("Deleting interface %+v", names.BridgeName)
	}
	return nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	netID := r.NetworkID
	return d.deleteNics(netID)
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) error {
	log.Debugf("Create endpoint request: %+v", r)
	netID := r.NetworkID
	// get the links
	_, err := d.getLinks(netID)
	if err != nil {
		return err
	}
	return nil
}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
	log.Debugf("Delete endpoint request: %+v", r)
	if d.allow_empty {
		return nil
	}

	netID := r.NetworkID

	links, err := d.getLinks(netID)
	if err != nil {
		return err
	}
	VxlanIndex := links.Vxlan.LinkAttrs.Index
	BridgeIndex := links.Bridge.LinkAttrs.Index

	allLinks, err := netlink.LinkList()
	if err != nil {
		return err
	}

	for i := range allLinks {
		if allLinks[i].Attrs().Index != VxlanIndex && allLinks[i].Attrs().MasterIndex == BridgeIndex {
			return nil
		}
	}

	log.Debugf("No interfaces attached to bridge: deleting vxlan and bridge interfaces.")
	return d.deleteNics(netID)
}

func (d *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
	res := &network.InfoResponse{
		Value: make(map[string]string),
	}
	return res, nil
}

func (d *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
	netID := r.NetworkID
	// get the links
	links, err := d.getLinks(netID)
	if err != nil {
		return nil, err
	}
	// create and attach local name to the bridge
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth_" + r.EndpointID[:5],
		MTU: links.Bridge.LinkAttrs.MTU },
		PeerName:  "ethc" + r.EndpointID[:5],
	}
	if err := netlink.LinkAdd(veth); err != nil {
		log.Errorf("failed to create the veth pair named: [ %v ] error: [ %s ] ", veth, err)
		return nil, err
	}

	// bring up the veth pair
	err = netlink.LinkSetUp(veth)
	if err != nil {
		log.Warnf("Error enabling  Veth local iface: [ %v ]", veth)
		return nil, err
	}
	
	// add veth to bridge
	err = netlink.LinkSetMaster(veth, links.Bridge)
	if err != nil {
		return nil, err
	}

	// SrcName gets renamed to DstPrefix + ID on the container iface
	gateway, err := getGateway(netID, *d.docker)
	if err != nil {
		return nil, err
	}
	res := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   veth.PeerName,
			DstPrefix: "eth",
		},
		Gateway: gateway,
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
