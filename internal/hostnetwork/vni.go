package hostnetwork

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
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

	hostVeth, peVeth, err := setupVeth(params.Name)
	if err != nil {
		return err
	}
	err = assignIPToInterface(hostVeth, params.VethHostIP)
	if err != nil {
		return err
	}

	err = netlink.LinkSetUp(hostVeth)
	if err != nil {
		return fmt.Errorf("could not set link up for host leg %s: %v", hostVeth, err)
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
		err = netlink.LinkSetUp(peVeth)
		if err != nil {
			return fmt.Errorf("could not set link up for host leg %s: %v", hostVeth, err)
		}

		vrf, err := setupVRF(params.Name)
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
