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
		loopback, err := netlink.LinkByName("lo")
		if err != nil {
			return fmt.Errorf("assignVTEPToLoopback: failed to get lo")
		}
		err = assignIPToInterface(loopback, params.VtepIP)
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}
