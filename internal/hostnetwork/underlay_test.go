package hostnetwork

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	externalInterfaceIP   = "192.170.0.9/24"
	underlayTestNS        = "underlaytest"
	underlayTestInterface = "testunderlayext"
)

func TestUnderlay(t *testing.T) {
	cleanTest(t, underlayTestNS)

	toMove := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: underlayTestInterface,
		},
	}
	err := netlink.LinkAdd(toMove)
	if err != nil {
		t.Fatalf("failed to create interface %s: %v", toMove.Name, err)
	}

	err = assignIPToInterface(toMove, externalInterfaceIP)
	if err != nil {
		t.Fatalf("failed to assign ip to current interface: %v", err)
	}

	_, newNs := createTestNS(t, underlayTestNS)

	params := UnderlayParams{
		MainNic:  underlayTestInterface,
		VtepIP:   "192.168.1.1/32",
		TargetNS: underlayTestNS,
	}
	err = SetupUnderlay(params)
	if err != nil {
		t.Fatalf("failed to setup underlay %s", err)
	}
	_ = inNamespace(newNs, func() error {
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
			if l.Attrs().Name == underlayTestInterface {
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

		return nil
	})
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

func cleanTest(t *testing.T, namespace string) {
	t.Helper()
	err := netns.DeleteNamed(namespace)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to delete ns: %v", err)
	}
	links, err := netlink.LinkList()
	if err != nil {
		t.Fatalf("failed to list links: %v", err)
	}
	for _, l := range links {
		if strings.HasPrefix(l.Attrs().Name, "test") {
			err := netlink.LinkDel(l)
			if err != nil {
				t.Fatalf("failed remove link %s: %v", l.Attrs().Name, err)
			}
		}
	}
}

func createTestNS(t *testing.T, testNs string) (netns.NsHandle, netns.NsHandle) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	currentNs, err := netns.Get()
	if err != nil {
		t.Fatalf("failed to create new ns %s", err)
	}

	newNs, err := netns.NewNamed(testNs)
	if err != nil {
		t.Fatalf("failed to create new ns %s", err)
	}

	t.Cleanup(func() {
		err := currentNs.Close()
		if err != nil {
			t.Fatalf("failed to close current ns")
		}
		err = newNs.Close()
		if err != nil {
			t.Fatalf("failed to close new ns")
		}
		cleanTest(t, underlayTestInterface)
	})

	err = netns.Set(currentNs)
	if err != nil {
		t.Fatalf("failed to restore to current ns %s", err)
	}
	return currentNs, newNs
}
