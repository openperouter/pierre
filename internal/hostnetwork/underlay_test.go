package hostnetwork

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	externalInterfaceIP       = "192.170.0.9/24"
	underlayTestNS            = "underlaytest"
	underlayTestInterface     = "testundfirst"
	underlayTestInterfaceEdit = "testundsec"
	externalInterfaceEditIP   = "192.170.0.10/24"
)

func TestUnderlay(t *testing.T) {
	cleanTest(t, underlayTestNS)

	setup := func() netns.NsHandle {
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
		toEdit := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: underlayTestInterfaceEdit,
			},
		}
		err = netlink.LinkAdd(toEdit)
		if err != nil {
			t.Fatalf("failed to create interface %s: %v", toEdit.Name, err)
		}

		err = assignIPToInterface(toEdit, externalInterfaceEditIP)
		if err != nil {
			t.Fatalf("failed to assign ip to current interface: %v", err)
		}

		_, testNs := createTestNS(t, underlayTestNS)
		return testNs
	}

	t.Run("test single underlay", func(t *testing.T) {
		cleanTest(t, underlayTestNS)
		testNs := setup()
		params := UnderlayParams{
			MainNic:  underlayTestInterface,
			VtepIP:   "192.168.1.1/32",
			TargetNS: underlayTestNS,
		}
		err := SetupUnderlay(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup underlay %s", err)
		}

		validateUnderlay(t, testNs, externalInterfaceIP, params)
	})

	t.Run("test underlay is idempotent", func(t *testing.T) {
		cleanTest(t, underlayTestNS)
		testNs := setup()
		params := UnderlayParams{
			MainNic:  underlayTestInterface,
			VtepIP:   "192.168.1.1/32",
			TargetNS: underlayTestNS,
		}
		err := SetupUnderlay(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup underlay %s", err)
		}
		err = SetupUnderlay(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup underlay %s", err)
		}

		validateUnderlay(t, testNs, externalInterfaceIP, params)
	})

	t.Run("test underlay changes primary interface and vtep", func(t *testing.T) {
		cleanTest(t, underlayTestNS)
		testNs := setup()

		params := UnderlayParams{
			MainNic:  underlayTestInterface,
			VtepIP:   "192.168.1.1/32",
			TargetNS: underlayTestNS,
		}
		err := SetupUnderlay(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup underlay %s", err)
		}

		params.MainNic = underlayTestInterfaceEdit
		params.VtepIP = "192.168.1.2/32"

		err = SetupUnderlay(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup underlay %s", err)
		}

		validateUnderlay(t, testNs, externalInterfaceEditIP, params)
	})
	cleanTest(t, underlayTestNS)
}

func validateUnderlay(t *testing.T, ns netns.NsHandle, ipToValidate string, params UnderlayParams) {
	_ = inNamespace(ns, func() error {
		links, err := netlink.LinkList()
		if err != nil {
			t.Fatalf("failed to list links %v", err)
		}
		loopbackFound := false
		mainNicFound := false
		for _, l := range links {
			if l.Attrs().Name == UnderlayLoopback {
				loopbackFound = true
				validateIP(t, l, params.VtepIP)
			}
			if l.Attrs().Name == params.MainNic {
				mainNicFound = true
				validateIP(t, l, ipToValidate)
				validateIP(t, l, underlayNicSpecialAddr)
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
		if strings.HasPrefix(l.Attrs().Name, "test") ||
			strings.HasPrefix(l.Attrs().Name, PEVethPrefix) ||
			strings.HasPrefix(l.Attrs().Name, HostVethPrefix) {
			err := netlink.LinkDel(l)
			if err != nil {
				t.Fatalf("failed remove link %s: %v", l.Attrs().Name, err)
			}
		}
	}
	loopback, err := netlink.LinkByName(UnderlayLoopback)
	if errors.As(err, &netlink.LinkNotFoundError{}) {
		return
	}
	if err != nil {
		t.Fatalf("failed to find link %s: %v", UnderlayLoopback, err)
	}
	err = netlink.LinkDel(loopback)
	if err != nil {
		t.Fatalf("failed remove link %s: %v", UnderlayLoopback, err)
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
