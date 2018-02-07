package vxlan

import (
	"errors"
	"fmt"
	"net"
	gonet "net"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/vishvananda/netlink"

	dockerclient "github.com/docker/engine-api/client"
	dockertypes "github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

type Driver struct {
	network.Driver
	scope        string
	vtepdev      string
	localGateway bool
	docker       *dockerclient.Client
}

func NewDriver(scope string, vtepdev string, localGW bool) (*Driver, error) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	docker, err := dockerclient.NewClient("unix:///var/run/docker.sock", "v1.23", nil, defaultHeaders)
	if err != nil {
		log.Errorf("Error connecting to docker socket: %v", err)
		return nil, err
	}
	d := &Driver{
		scope:        scope,
		vtepdev:      vtepdev,
		localGateway: localGW,
		docker:       docker,
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
	VxlanName string
}

func getIntNames(netID string, docker *dockerclient.Client) (*intNames, error) {
	net, err := docker.NetworkInspect(context.Background(), netID)
	if err != nil {
		log.Errorf("Error getting networks: %v", err)
		return nil, err
	}

	names := &intNames{}

	if net.Driver != "vxlan" {
		log.Errorf("Network %v is not a vxlan network", netID)
		return nil, errors.New("Not a vxlan network")
	}

	names.VxlanName = "vx_" + netID[:12]

	// get interface names from options first
	for k, v := range net.Options {
		if k == "vxlanName" {
			names.VxlanName = v
		}
	}

	return names, nil
}

func getGateway(netID string, docker *dockerclient.Client) (string, string, error) {
	net, err := docker.NetworkInspect(context.Background(), netID)
	if err != nil {
		log.Errorf("Error inspecting network: %v", err)
		return "", "", err
	}

	for _, config := range net.IPAM.Config {
		gw := config.Gateway
		sn := config.Subnet
		if gw != "" && sn != "" {
			return gw, sn, nil
		}
	}

	return "", "", nil
}

type intLinks struct {
	Vxlan *netlink.Vxlan
}

// this function gets netlink devices or creates them if they don't exist
func (d *Driver) getLinks(netID string) (*intLinks, error) {
	docker := d.docker
	net, err := docker.NetworkInspect(context.Background(), netID)
	if err != nil {
		log.Errorf("Error inspecting network: %v", err)
		return nil, err
	}

	if net.Driver != "vxlan" {
		log.Errorf("Network %v is not a vxlan network", netID)
		return nil, errors.New("Not a vxlan network")
	}

	names, err := getIntNames(netID, docker)
	if err != nil {
		log.Errorf("Error getting interface names: %v", err)
		return nil, err
	}

	// get or create links
	var vxlan *netlink.Vxlan
	vxlanlink, err := netlink.LinkByName(names.VxlanName)
	if err == nil {
		vxlan = &netlink.Vxlan{
			LinkAttrs: *vxlanlink.Attrs(),
		}
	} else {
		vxlan, err = d.createVxLan(names.VxlanName, &net)
		if err != nil {
			log.Errorf("Error creating vxlan: %v", err)
			return nil, err
		}
	}

	links := &intLinks{
		Vxlan: vxlan,
	}

	return links, nil
}

func (d *Driver) createVxLan(vxlanName string, nr *dockertypes.NetworkResource) (*netlink.Vxlan, error) {
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: vxlanName,
		},
	}

	// Parse interface options
	for k, v := range nr.Options {
		if k == "vxlanMTU" {
			MTU, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting MTU to int: %v", err)
				return nil, err
			}
			vxlan.LinkAttrs.MTU = MTU
		}
		if k == "vxlanHardwareAddr" {
			HardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				log.Errorf("Error parsing mac: %v", err)
				return nil, err
			}
			vxlan.LinkAttrs.HardwareAddr = HardwareAddr
		}
		if k == "vxlanTxQLen" {
			TxQLen, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting TxQLen to int: %v", err)
				return nil, err
			}
			vxlan.LinkAttrs.TxQLen = TxQLen
		}
		if k == "VxlanId" {
			log.Debugf("VxlanID: %+v", v)
			VxlanID, err := strconv.ParseInt(v, 0, 32)
			if err != nil {
				log.Errorf("Error converting VxlanId to int: %v", err)
				return nil, err
			}
			log.Debugf("VxlanID: %+v", VxlanID)
			log.Debugf("int(VxlanID): %+v", int(VxlanID))
			vxlan.VxlanId = int(VxlanID)
		}
		if k == "VtepDev" {
			vtepDev, err := netlink.LinkByName(v)
			if err != nil {
				log.Errorf("Error getting VtepDev interface: %v", err)
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
				log.Errorf("Error converting TTL to int: %v", err)
				return nil, err
			}
			vxlan.TTL = TTL
		}
		if k == "TOS" {
			TOS, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting TOS to int: %v", err)
				return nil, err
			}
			vxlan.TOS = TOS
		}
		if k == "Learning" {
			Learning, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting Learning to bool: %v", err)
				return nil, err
			}
			vxlan.Learning = Learning
		}
		if k == "Proxy" {
			Proxy, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting Proxy to bool: %v", err)
				return nil, err
			}
			vxlan.Proxy = Proxy
		}
		if k == "RSC" {
			RSC, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting RSC to bool: %v", err)
				return nil, err
			}
			vxlan.RSC = RSC
		}
		if k == "L2miss" {
			L2miss, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting L2miss to bool: %v", err)
				return nil, err
			}
			vxlan.L2miss = L2miss
		}
		if k == "L3miss" {
			L3miss, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting L3miss to bool: %v", err)
				return nil, err
			}
			vxlan.L3miss = L3miss
		}
		if k == "NoAge" {
			NoAge, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting NoAge to bool: %v", err)
				return nil, err
			}
			vxlan.NoAge = NoAge
		}
		if k == "GBP" {
			GBP, err := strconv.ParseBool(v)
			if err != nil {
				log.Errorf("Error converting GBP to bool: %v", err)
				return nil, err
			}
			vxlan.GBP = GBP
		}
		if k == "Age" {
			Age, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting Age to int: %v", err)
				return nil, err
			}
			vxlan.Age = Age
		}
		if k == "Limit" {
			Limit, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting Limit to int: %v", err)
				return nil, err
			}
			vxlan.Limit = Limit
		}
		if k == "Port" {
			Port, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting Port to int: %v", err)
				return nil, err
			}
			vxlan.Port = Port
		}
		if k == "PortLow" {
			PortLow, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting PortLow to int: %v", err)
				return nil, err
			}
			vxlan.PortLow = PortLow
		}
		if k == "PortHigh" {
			PortHigh, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting PortHigh to int: %v", err)
				return nil, err
			}
			vxlan.PortHigh = PortHigh
		}
	}

	if d.vtepdev != "" {
		vtepDev, err := netlink.LinkByName(d.vtepdev)
		if err != nil {
			log.Errorf("Error getting vtepdev interface: %v", err)
			return nil, err
		}
		vxlan.VtepDevIndex = vtepDev.Attrs().Index
	}

	err := netlink.LinkAdd(vxlan)
	if err != nil {
		log.Errorf("Error adding vxlan interface: %v", err)
		return nil, err
	}

	// Parse interface options
	for k, v := range nr.Options {
		if k == "vxlanHardwareAddr" {
			hardwareAddr, err := gonet.ParseMAC(v)
			if err != nil {
				log.Errorf("Error parsing mac address: %v", err)
				return nil, err
			}
			err = netlink.LinkSetHardwareAddr(vxlan, hardwareAddr)
			if err != nil {
				log.Errorf("Error setting mac address: %v", err)
				return nil, err
			}
		}
		if k == "vxlanMTU" {
			mtu, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Error converting MTU to int: %v", err)
				return nil, err
			}
			err = netlink.LinkSetMTU(vxlan, mtu)
			if err != nil {
				log.Errorf("Error setting MTU: %v", err)
				return nil, err
			}
		}
	}

	// bring interfaces up
	err = netlink.LinkSetUp(vxlan)
	if err != nil {
		log.Errorf("Error bringing up vxlan: %v", err)
		return nil, err
	}

	//if localGateway is set
	if d.localGateway {
		gw, sn, err := getGateway(nr.ID, d.docker)
		if err != nil {
			log.Errorf("Error getting gateway: %v", err)
			return nil, err
		}
		gwip := net.ParseIP(gw)
		if gwip == nil {
			log.Errorf("Failed to parse Gateway IP: %v", gw)
			return nil, err
		}
		_, nw, err := net.ParseCIDR(sn)
		if err != nil {
			log.Errorf("Failed to parse subnet: %v", err)
			return nil, err
		}

		//create macvlan interface
		mvlName := "gwmvl_" + strconv.Itoa(vxlan.LinkAttrs.Index)
		mvl, err := makeMacvlan(mvlName, vxlan.LinkAttrs.Index)
		if err != nil {
			log.Errorf("Failed to create host macvlan on vxlan %v: %v", vxlan.LinkAttrs.Name, err)
			return nil, err
		}

		hmvlAddr := &net.IPNet{IP: gwip, Mask: nw.Mask}
		hostNla := &netlink.Addr{
			IPNet: hmvlAddr,
			Label: "",
		}

		//add gateway address to it
		err = netlink.AddrAdd(mvl, hostNla)
		if err != nil {
			log.Errorf("Failed to add gateway address to host macvlan on vxlan %v: %v", vxlan.LinkAttrs.Name, err)
			//delete macvlan device
			err2 := deleteMacvlan(mvlName)
			if err2 != nil {
				log.Errorf("Failed to delete macvlan %v after failing to add an address to it. %v", mvlName, err2)
			}
			return nil, err
		}
	}

	return vxlan, nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	log.Debugf("Create network request: %+v", r)
	return nil
}

func (d *Driver) deleteNics(netID string) error {
	names, err := getIntNames(netID, d.docker)
	if err != nil {
		return err
	}

	vxlan, err := netlink.LinkByName(names.VxlanName)
	if err != nil {
		return nil
	}

	//macvlan is implicitly deleted
	err = netlink.LinkDel(vxlan)
	if err != nil {
		return err
	}
	log.Debugf("Deleting interface %+v", names.VxlanName)

	return nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	netID := r.NetworkID
	return d.deleteNics(netID)
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	log.Debugf("Create endpoint request: %+v", r)
	netID := r.NetworkID
	// get the links
	_, err := d.getLinks(netID)
	if err != nil {
		log.Errorf("Error getting links: %v", err)
		return nil, err
	}
	return &network.CreateEndpointResponse{}, nil
}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
	log.Debugf("Delete endpoint request: %+v", r)

	linkName := "macvlan_" + r.EndpointID[:7]
	err := deleteMacvlan(linkName)
	if err != nil {
		log.Errorf("Failed to delete macvlan %v", linkName)
		return err
	}

	// Asynchronously check and remove the vxlan interface if nothing else is using it.
	go d.cleanup(r.NetworkID)
	return nil
}

func (d *Driver) cleanup(netID string) {
	links, err := d.getLinks(netID)
	if err != nil {
		log.Errorf("Error getting links: %v", err)
		return
	}
	VxlanIndex := links.Vxlan.LinkAttrs.Index

	allLinks, err := netlink.LinkList()
	if err != nil {
		log.Errorf("Error getting all links: %v", err)
		return
	}

	// Do nothing if other interfaces are slaves of the vxlan interface
	for _, link := range allLinks {
		//if localGateway is set and this link is a gwmvl_ link
		if d.localGateway && strings.HasPrefix(link.Attrs().Name, "gwmvl_") {
			continue
		}
		if link.Attrs().MasterIndex == VxlanIndex {
			log.Debugf("Interface still attached to vxlan: %v", link)
			return
		}
	}

	// Do nothing if there are other containers in this network
	netResource, err := d.docker.NetworkInspect(context.Background(), netID)
	if err != nil {
		log.Errorf("Error inspecting network: %v", err)
		return
	}
	netName := netResource.Name

	containers, err := d.docker.ContainerList(context.Background(), dockertypes.ContainerListOptions{})
	if err != nil {
		log.Errorf("Error getting containers: %v", err)
		return
	}
	for _, container := range containers {
		if _, ok := container.NetworkSettings.Networks[netName]; ok {
			log.Debugf("Other containers are still connected to this network")
			return
		}
	}

	log.Debugf("No interfaces attached to vxlan: deleting vxlan interface.")
	err = d.deleteNics(netID)
	if err != nil {
		log.Errorf("Error deleting nics: %v", err)
	}
	return
}

func (d *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
	res := &network.InfoResponse{
		Value: make(map[string]string),
	}
	return res, nil
}

func (d *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
	log.Debugf("Join endpoint request: %+v", r)
	netID := r.NetworkID
	// get the links
	links, err := d.getLinks(netID)
	if err != nil {
		log.Errorf("Error getting link: %v", err)
		return nil, err
	}

	mvlName := "macvlan_" + r.EndpointID[:7]
	_, err = makeMacvlan(mvlName, links.Vxlan.LinkAttrs.Index)
	if err != nil {
		log.Errorf("Error creating macvlan: %v", err)
		return nil, err
	}

	gateway, _, err := getGateway(netID, d.docker)
	if err != nil {
		log.Errorf("Error getting gateway: %v", err)
		return nil, err
	}
	res := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   mvlName,
			DstPrefix: "eth",
		},
		Gateway: gateway,
	}
	log.Debugf("Join endpoint %s:%s to %s", r.NetworkID, r.EndpointID, r.SandboxKey)
	return res, nil
}

func (d *Driver) Leave(r *network.LeaveRequest) error {
	log.Debugf("Leave endpoint request: %+v", r)
	return nil

}

// The vxlan driver will not expose ports, just respond empty.
func (d *Driver) ProgramExternalConnectivity(r *network.ProgramExternalConnectivityRequest) error {
	log.Debugf("Program External Connectivity request: %+v", r)
	return nil
}

func (d *Driver) RevokeExternalConnectivity(r *network.RevokeExternalConnectivityRequest) error {
	log.Debugf("Revoke External Connectivity request: %+v", r)
	return nil
}

func (d *Driver) DiscoverNew(r *network.DiscoveryNotification) error {
	log.Debugf("Discover New request: %+v", r)
	return nil
}

func (d *Driver) DiscoverDelete(r *network.DiscoveryNotification) error {
	log.Debugf("Discover Delete request: %+v", r)
	return nil
}

func makeMacvlan(name string, parent int) (*netlink.Macvlan, error) {
	log.Debugf("Creating new macvlan: %v", name)
	// Create a macvlan link
	mvl := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        name,
			ParentIndex: parent,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}
	if err := netlink.LinkAdd(mvl); err != nil {
		log.Errorf("Error adding link: %v", err)
		return nil, err
	}

	return mvl, nil
}

func deleteMacvlan(name string) error {
	// Delete the macvlan interface
	macvlanLink, err := netlink.LinkByName(name)
	if err != nil {
		log.Errorf("Error getting macvlan (%v) link: %v", name, err)
		return err
	}
	// verify a parent interface isn't being deleted
	if macvlanLink.Attrs().ParentIndex == 0 {
		log.Errorf("interface (%v) does not appear to be a slave interface", name)
		return fmt.Errorf("interface %s does not appear to be a slave device: %v", name, err)
	}
	// delete the macvlan slave device
	if err := netlink.LinkDel(macvlanLink); err != nil {
		log.Errorf("Error deleting link: %v", err)
		return err
	}

	log.Debugf("Deleted macvlan: %s", name)
	return nil
}
