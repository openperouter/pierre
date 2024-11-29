package hostnetwork

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	externalInterfaceIP = "192.170.0.9/24"
	vniTestNS           = "vnitest"
	vniTestInterface    = "vniexternal"
)

func TestUnderlay(t *testing.T) {
	Clean(t)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	toMove := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: vniTestInterface,
		},
	}
	err := netlink.LinkAdd(toMove)
	if err != nil {
		t.Fatalf("failed to create interface %s: %v", toMove.Attrs().Name, err)
	}

	err = assignIPToInterface(toMove, externalInterfaceIP)
	if err != nil {
		t.Fatalf("failed to assign ip to current interface: %v", err)
	}

	_, newNs := createTestNS(t, vniTestNS)

	params := UnderlayParams{
		MainNic:  vniTestInterface,
		VtepIP:   "192.168.1.1/32",
		TargetNS: vniTestNS,
	}
	err = SetupUnderlay(params)
	if err != nil {
		t.Fatalf("failed to setup underlay %s", err)
	}
	err = netns.Set(newNs)
	if err != nil {
		t.Fatalf("failed to switch to pe ns %v", err)
	}
	links, err := netlink.LinkList()
	if err != nil {
		t.Fatalf("failed to list links %v", err)
	}
	loopbackFound := false
	mainNicFound := false
	for _, l := range links {
		if l.Attrs().Name == "lo" {
			loopbackFound = true
			validateIP(t, l, params.VtepIP)
		}
		if l.Attrs().Name == vniTestInterface {
			mainNicFound = true
			validateIP(t, l, externalInterfaceIP)
		}

	}
	if !loopbackFound {
		t.Fatalf("failed to find loopback in ns, links %v", links)
	}
	if !mainNicFound {
		t.Fatalf("failed to find loopback in ns, links %v", links)
	}
}

func validateIP(t *testing.T, l netlink.Link, address string) {
	t.Helper()
	addresses, err := netlink.AddrList(l, netlink.FAMILY_ALL)
	if err != nil {
		t.Fatalf("failed to list addresses for %s: %v", l.Attrs().Name, err)
	}
	found := false
	for _, a := range addresses {
		if a.IPNet.String() == address {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("failed to find address %s for %s: %v", address, l.Attrs().Name, addresses)
	}
}

func Clean(t *testing.T) {
	t.Helper()
	err := netns.DeleteNamed(vniTestNS)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to delete ns: %v", err)
	}
	toDel, err := netlink.LinkByName(vniTestInterface)
	if errors.As(err, &netlink.LinkNotFoundError{}) {
		return
	}
	if err != nil {
		t.Fatalf("failed to get link %s: %v", vniTestInterface, err)
	}
	err = netlink.LinkDel(toDel)
	if err != nil {
		t.Fatalf("failed to delete link %s: %v", vniTestInterface, err)
	}
}

func createTestNS(t *testing.T, testNs string) (netns.NsHandle, netns.NsHandle) {
	t.Helper()
	currentNs, err := netns.Get()
	if err != nil {
		t.Fatalf("failed to create new ns %s", err)
	}

	newNs, err := netns.NewNamed(vniTestNS)
	if err != nil {
		t.Fatalf("failed to create new ns %s", err)
	}

	t.Cleanup(func() {
		currentNs.Close()
		newNs.Close()
		Clean(t)
	})

	err = netns.Set(currentNs)
	if err != nil {
		t.Fatalf("failed to restore to current ns %s", err)
	}
	return currentNs, newNs
}
