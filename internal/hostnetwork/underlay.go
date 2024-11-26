package hostnetwork

import (
	"fmt"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type UnderlayParams struct {
	MainNic  string
	VtepIP   string
	TargetNS string
}

func SetupUnderlay(params UnderlayParams) error {
	ns, err := netns.GetFromPath(params.TargetNS)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to find network namespace %s: %w", params.TargetNS, err)
	}
	defer ns.Close()

	err = moveNicToNamespace(params.MainNic, ns)
	if err != nil {
		return err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current network namespace
	origns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to get current network namespace")
	}
	defer origns.Close()

	err = netns.Set(ns)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to set current network namespace to %s", ns.String())
	}
	defer func() { netns.Set(origns) }()

	err = assignVTEPToLoopback(params.VtepIP)
	if err != nil {
		return err
	}

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

	err = netlink.LinkSetNsFd(link, int(ns))
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to move %s to network namespace %s: %w", link.Attrs().Name, ns.String(), err)
	}
	return nil
}
