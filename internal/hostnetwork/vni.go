package hostnetwork

import (
	"fmt"
	"runtime"

	"github.com/vishvananda/netns"
)

type vniParams struct {
	TargetNS string
}

func SetupVNI(params vniParams) error {
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
