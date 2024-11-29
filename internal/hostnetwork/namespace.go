package hostnetwork

import (
	"fmt"
	"runtime"

	"github.com/vishvananda/netns"
)

func inNamespace(ns netns.NsHandle, execInNamespace func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
	err = execInNamespace()
	if err != nil {
		return err
	}
	return nil
}
