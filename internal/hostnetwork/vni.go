package hostnetwork

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

type vniParams struct {
	Name       string
	TargetNS   string
	VTEPIP     string
	VethHostIP string
	VethNSIP   string
	VNI        int
	VXLanPort  int
}

func SetupVNI(params vniParams) error {
	ns, err := netns.GetFromName(params.TargetNS)
	if err != nil {
		return fmt.Errorf("SetupVNI: Failed to get network namespace %s", params.TargetNS)
	}

	hostVeth, peVeth, err := createVeth(params.Name)
	if err != nil {
		return err
	}
	err = assignIPToInterface(hostVeth, params.VethHostIP)
	if err != nil {
		return err
	}

	err = netlink.LinkSetNsFd(peVeth, int(ns))
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to move %s to network namespace %s: %w", peVeth.Attrs().Name, ns.String(), err)
	}

	err = inNamespace(ns, func() error {
		err = assignIPToInterface(peVeth, params.VethNSIP)
		if err != nil {
			return err
		}

		vrf, err := createVRF(params.Name)
		if err != nil {
			return err
		}

		bridge, err := setupBridge(params, vrf)
		if err != nil {
			return err
		}

		err = setupVXLan(params, bridge)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

func addVeth(name string) (netlink.Link, netlink.Link, error) {
	hostSide := name + "host"
	peSide := name + "pe"

	vethHost := &netlink.Veth{}
	link, err := netlink.LinkByName(hostSide)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		vethHost, err = createVeth(name)
		if err != nil {
			return nil, nil, err
		}
	}

	vethHost, ok := link.(*netlink.Veth)
	if !ok {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
	}
	if vethHost.PeerName != peSide {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
	}

	err = netlink.LinkSetUp(vethHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not set veth %s up: %w", name, err)
	}
	peerIndex, err := netlink.VethPeerIndex(vethHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth for %s: %w", name, err)
	}
	vethPE, err := netlink.LinkByIndex(peerIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth by index for %s: %w", name, err)
	}
	return vethHost, vethPE, nil
}

func createVeth(name string) (*netlink.Veth, error) {
	hostSide := name + "host"
	peSide := name + "pe"
	vethHost := &netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: hostSide}, PeerName: peSide}
	err := netlink.LinkAdd(vethHost)
	if err != nil {
		return nil, fmt.Errorf("could not add veth %s: %w", name, err)
	}
	return vethHost, nil

}

func setupBridge(params vniParams, vrf *netlink.Vrf) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		Name:        fmt.Sprintf("br%d", params.VNI),
		MasterIndex: vrf.Index,
	}}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return nil, fmt.Errorf("could not create bridge for VNI %d", params.VNI)
	}
	err = addrGenModeNone(bridge)
	if err != nil {
		return nil, fmt.Errorf("failed to set addr_gen_mode to 1 for %s: %w", bridge.Name, err)
	}

	return bridge, nil
}

func setupVXLan(params vniParams, bridge *netlink.Bridge) error {
	loopback, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("failed to get loopback by name: %w", err)
	}

	vxlan := &netlink.Vxlan{LinkAttrs: netlink.LinkAttrs{
		Name:        fmt.Sprintf("vni%d", params.VNI),
		MasterIndex: bridge.Index,
	},
		VxlanId:      params.VNI,
		Port:         params.VXLanPort,
		Learning:     false,
		SrcAddr:      net.ParseIP(params.VTEPIP),
		VtepDevIndex: loopback.Attrs().Index,
	}
	err = netlink.LinkAdd(vxlan)
	if err != nil {
		return fmt.Errorf("failed to create vxlan %s: %w", vxlan.Name, err)
	}
	err = addrGenModeNone(vxlan)
	if err != nil {
		return fmt.Errorf("failed to set addr_gen_mode to 1 for %s: %w", vxlan.Name, err)
	}
	err = setNeighSuppression(vxlan)
	if err != nil {
		return fmt.Errorf("failed to set neigh suppression for %s: %w", vxlan.Name, err)
	}
	return nil
}

func assignIPToInterface(link netlink.Link, address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("assignIPToInterface: failed to parse address %s for interface %s", address, link.Attrs().Name)
	}
	addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("assignIPToInterface: failed to list addresses for interface %s", link.Attrs().Name)
	}
	for _, a := range addresses {
		if a.IPNet.String() == address {
			return nil
		}
	}
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return fmt.Errorf("assignIPToInterface: failed to add address %s to interface %s, err %v", address, link.Attrs().Name, err)
	}
	return nil
}

func addrGenModeNone(l netlink.Link) error {
	fileName := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/addr_gen_mode", l.Attrs().Name)
	file, err := os.OpenFile(fileName, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("addrGenModeNone: error opening file: %w", err)
	}
	defer file.Close()
	if _, err := fmt.Fprintf(file, "%s\n", "1"); err != nil {
		return fmt.Errorf("addrGenModeNone: error writing to file: %w", err)
	}
	return nil
}

func setNeighSuppression(link netlink.Link) error {
	req := nl.NewNetlinkRequest(unix.RTM_SETLINK, unix.NLM_F_ACK)

	msg := nl.NewIfInfomsg(unix.AF_BRIDGE)
	msg.Index = int32(link.Attrs().Index)
	req.AddData(msg)

	br := nl.NewRtAttr(unix.IFLA_PROTINFO|unix.NLA_F_NESTED, nil)
	br.AddRtAttr(32, []byte{1})
	req.AddData(br)
	_, err := req.Execute(unix.NETLINK_ROUTE, 0)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	return nil
}
