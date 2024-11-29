package hostnetwork

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type UnderlayParams struct {
	MainNic  string
	VtepIP   string
	TargetNS string
}

func SetupUnderlay(params UnderlayParams) error {
	ns, err := netns.GetFromName(params.TargetNS)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to find network namespace %s: %w", params.TargetNS, err)
	}
	defer ns.Close()

	err = moveNicToNamespace(params.MainNic, ns)
	if err != nil {
		return err
	}

	inNamespace(ns, func() error {
		err = assignVTEPToLoopback(params.VtepIP)
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func assignVTEPToLoopback(address string) error {
	loopback, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("assignVTEPToLoopback: failed to get lo")
	}
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("assignVTEPToLoopback: failed to parse address %s", address)
	}

	err = netlink.AddrAdd(loopback, addr)
	if err != nil {
		return fmt.Errorf("assignVTEPToLoopback: failed to parse address %s", address)
	}
	return nil
}

func moveNicToNamespace(nic string, ns netns.NsHandle) error {
	link, err := netlink.LinkByName(nic)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to find link %s: %w", nic, err)
	}

	addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to get addresses for nic %s: %w", link.Attrs().Name, err)
	}

	err = netlink.LinkSetNsFd(link, int(ns))
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to move %s to network namespace %s: %w", link.Attrs().Name, ns.String(), err)
	}
	inNamespace(ns, func() error {
		for _, a := range addresses {
			err := netlink.AddrAdd(link, &a)
			if err != nil {
				return fmt.Errorf("moveNicToNamespace: Failed to add address %s to %s", a, link)
			}
		}
		return nil
	})
	return nil
}
