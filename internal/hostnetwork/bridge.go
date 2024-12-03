package hostnetwork

import (
	"errors"
	"fmt"

	"github.com/vishvananda/netlink"
)

func setupBridge(params VNIParams, vrf *netlink.Vrf) (*netlink.Bridge, error) {

	name := bridgeName(params.VNI)
	link, err := netlink.LinkByName(name)
	if err != nil && errors.As(err, &netlink.LinkNotFoundError{}) {
		link, err = createBridge(name, vrf.Index)
		if err != nil {
			return nil, fmt.Errorf("failed to create bridge %s: %w", name, err)
		}
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
		bridge, err = createBridge(name, vrf.Index)
		if err != nil {
			return nil, fmt.Errorf("failed to create bridge %s: %w", name, err)
		}
	}

	err = addrGenModeNone(bridge)
	if err != nil {
		return nil, fmt.Errorf("failed to set addr_gen_mode to 1 for %s: %w", bridge.Name, err)
	}

	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return nil, fmt.Errorf("could not set link up for bridge %s: %v", name, err)
	}
	return bridge, nil
}

func createBridge(name string, vrfIndex int) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		Name:        name,
		MasterIndex: vrfIndex,
	}}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return nil, fmt.Errorf("could not create bridge %s", name)
	}

	return bridge, nil
}

func bridgeName(vni int) string {
	return fmt.Sprintf("br%d", vni)
}
