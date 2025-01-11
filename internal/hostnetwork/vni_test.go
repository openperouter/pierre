package hostnetwork

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const testNSName = "testns"

// TODO sleeps

func TestVNI(t *testing.T) {
	cleanTest(t, testNSName)
	setup := func() netns.NsHandle {
		_, testNS := createTestNS(t, testNSName)
		setupLoopback(t, testNS)
		return testNS
	}

	t.Run("single vni", func(t *testing.T) {
		testNS := setup()
		t.Cleanup(func() {
			cleanTest(t, testNSName)
		})

		params := VNIParams{
			VRF:        "testred",
			TargetNS:   testNSName,
			VTEPIP:     "192.170.0.9/32",
			VethHostIP: "192.168.9.1/32",
			VethNSIP:   "192.168.9.0/32",
			VNI:        100,
			VXLanPort:  4789,
		}

		err := SetupVNI(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup vni: %v", err)
		}

		time.Sleep(4 * time.Second)
		validateHostLeg(t, params)

		_ = inNamespace(testNS, func() error {
			validateNS(t, params)
			return nil
		})

	})

	t.Run("multiple vnis + cleanup", func(t *testing.T) {
		testNS := setup()
		t.Cleanup(func() {
			cleanTest(t, testNSName)
		})

		params := []VNIParams{
			{

				VRF:        "testred",
				TargetNS:   testNSName,
				VTEPIP:     "192.170.0.9/32",
				VethHostIP: "192.168.9.1/32",
				VethNSIP:   "192.168.9.0/32",
				VNI:        100,
				VXLanPort:  4789,
			},
			{
				VRF:        "testblue",
				TargetNS:   testNSName,
				VTEPIP:     "192.170.0.10/32",
				VethHostIP: "192.168.9.2/32",
				VethNSIP:   "192.168.9.3/32",
				VNI:        101,
				VXLanPort:  4789,
			},
		}
		for _, p := range params {
			err := SetupVNI(context.Background(), p)
			if err != nil {
				t.Fatalf("failed to setup vni: %v", err)
			}
			time.Sleep(5 * time.Second)
			validateHostLeg(t, p)
			_ = inNamespace(testNS, func() error {
				validateNS(t, p)
				return nil
			})
		}

		remaining := params[0]
		toDelete := params[1]
		err := RemoveNonConfiguredVNIs(testNS, []VNIParams{remaining})
		if err != nil {
			t.Fatalf("failed to remove non configured vnis: %v", err)
		}
		time.Sleep(5 * time.Second)
		validateHostLeg(t, remaining)
		_ = inNamespace(testNS, func() error {
			validateNS(t, remaining)
			return nil
		})

		hostSide, _ := vethLegsForVRF(toDelete.VRF)
		checkLinkdeleted(t, hostSide)
		validateVNIIsNotConfigured(t, toDelete)
	})

	t.Run("creation is idempotent", func(t *testing.T) {
		testNS := setup()
		t.Cleanup(func() {
			cleanTest(t, testNSName)
		})

		params := VNIParams{
			VRF:        "testred",
			TargetNS:   testNSName,
			VTEPIP:     "192.170.0.9/32",
			VethHostIP: "192.168.9.1/32",
			VethNSIP:   "192.168.9.0/32",
			VNI:        100,
			VXLanPort:  4789,
		}

		err := SetupVNI(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup vni: %v", err)
		}

		err = SetupVNI(context.Background(), params)
		if err != nil {
			t.Fatalf("failed to setup vni second time: %v", err)
		}

		time.Sleep(5 * time.Second)
		validateHostLeg(t, params)

		_ = inNamespace(testNS, func() error {
			validateNS(t, params)
			return nil
		})

	})
}

func validateHostLeg(t *testing.T, params VNIParams) {
	t.Helper()
	hostSide, _ := vethLegsForVRF(params.VRF)
	hostLegLink, err := netlink.LinkByName(hostSide)
	if err != nil {
		t.Fatalf("failed to get link by name: %v", err)
	}
	if hostLegLink.Attrs().OperState != netlink.OperUp {
		t.Fatalf("host leg %s is not up: %s", hostSide, hostLegLink.Attrs().OperState)
	}
	hasIP, err := interfaceHasIP(hostLegLink, params.VethHostIP)
	if err != nil {
		t.Fatalf("failed to undersand if host leg has ip: %v", err)
	}
	if !hasIP {
		addresses, _ := netlink.AddrList(hostLegLink, netlink.FAMILY_ALL)
		t.Fatalf("host leg doesn't have ip %s %v", params.VethHostIP, addresses)
	}
}

func validateNS(t *testing.T, params VNIParams) {
	t.Helper()
	loopback, err := netlink.LinkByName(UnderlayLoopback)
	if err != nil {
		t.Fatalf("failed to get loopback by name: %v", err)
	}

	vxlanLink, err := netlink.LinkByName(vxLanName(params.VNI))
	if err != nil {
		t.Fatalf("failed to get vxlan by name: %v", err)
	}
	vxlan := vxlanLink.(*netlink.Vxlan)
	if vxlan.OperState != netlink.OperUnknown { // todo should we even validate this?
		t.Fatalf("vxlan is not unknown: %s", vxlan.OperState)
	}
	addrGenModeNone, err := checkAddrGenModeNone(t, vxlan)
	if err != nil {
		t.Fatalf("failed to check addrGenModeNone %v", err)
	}
	if addrGenModeNone == false {
		t.Fatal("failed to check addrGenMode, expecting true")
	}

	vrfLink, err := netlink.LinkByName(params.VRF)
	if err != nil {
		t.Fatalf("failed to get vrf by name: %v", err)
	}
	vrf := vrfLink.(*netlink.Vrf)
	if vrf.OperState != netlink.OperUp {
		t.Fatalf("vrf is not up: %s", vrf.OperState)
	}

	bridgeLink, err := netlink.LinkByName(bridgeName(params.VNI))
	if err != nil {
		t.Fatalf("failed to get vxlan by name: %v", err)
	}
	bridge := bridgeLink.(*netlink.Bridge)
	if bridge.OperState != netlink.OperUp {
		t.Fatalf("bridge is not up: %s", bridge.OperState)
	}
	if bridge.MasterIndex != vrf.Index {
		t.Fatalf("bridge master is not vrf")
	}

	addrGenModeNone, err = checkAddrGenModeNone(t, bridge)
	if err != nil {
		t.Fatalf("failed to check addrGenModeNone %v", err)
	}
	if addrGenModeNone == false {
		t.Fatal("failed to check addrGenMode , expecting true")
	}

	err = checkVXLanConfigured(vxlan, bridge.Index, loopback.Attrs().Index, params)
	if err != nil {
		t.Fatalf("invalid vxlan %v", err)
	}

	_, peSide := vethLegsForVRF(params.VRF)
	peLegLink, err := netlink.LinkByName(peSide)
	if err != nil {
		t.Fatalf("failed to get peLegLink by name: %v", err)
	}
	if peLegLink.Attrs().OperState != netlink.OperUp {
		t.Fatalf("peLegLink is not up: %s", peLegLink.Attrs().OperState)
	}
	if peLegLink.Attrs().MasterIndex != vrf.Index {
		t.Fatalf("peLegLink master is not vrf")
	}

	hasIP, err := interfaceHasIP(peLegLink, params.VethNSIP)
	if err != nil {
		t.Fatalf("failed to undersand if pe leg has ip: %v", err)
	}
	if !hasIP {
		t.Fatalf("pe leg doesn't have ip %s", params.VethNSIP)
	}

	route, err := hostIPToRoute(vrf, params.VethHostIP, peLegLink)
	if err != nil {
		t.Fatalf("failed to convert host ip to route: %v", err)
	}
	isPresent, err := checkRouteIsPresent(route)
	if err != nil {
		t.Fatalf("failed to check if route is present: %v", err)
	}
	if !isPresent {
		t.Fatalf("route is not added")
	}
}

func checkLinkdeleted(t *testing.T, name string) {
	_, err := netlink.LinkByName(name)
	if err == nil {
		t.Fatalf("link %s is not deleted %s", name, err)
	}
	if !errors.As(err, &netlink.LinkNotFoundError{}) {
		t.Fatalf("failed to get link %s by name: %v", name, err)
	}
}

func validateVNIIsNotConfigured(t *testing.T, params VNIParams) {
	t.Helper()

	checkLinkdeleted(t, vxLanName(params.VNI))
	checkLinkdeleted(t, params.VRF)
	checkLinkdeleted(t, bridgeName(params.VNI))

	_, peSide := vethLegsForVRF(params.VRF)
	checkLinkdeleted(t, peSide)
}

func checkAddrGenModeNone(t *testing.T, l netlink.Link) (bool, error) {
	t.Helper()
	fileName := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/addr_gen_mode", l.Attrs().Name)
	addrGenMode, err := os.ReadFile(fileName)
	if err != nil {
		return false, err
	}
	if strings.Trim(string(addrGenMode), "\n") == "1" {
		return true, nil
	}
	return false, nil
}

func setupLoopback(t *testing.T, ns netns.NsHandle) {
	_ = inNamespace(ns, func() error {
		_, err := netlink.LinkByName(UnderlayLoopback)
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			loopback := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: UnderlayLoopback}}
			err = netlink.LinkAdd(loopback)
			if err != nil {
				t.Fatalf("setup lookback: failed to create %s", UnderlayLoopback)
			}
		}
		return nil
	})
}

func TestIPToRoute(t *testing.T) {
	vrf := &netlink.Vrf{
		LinkAttrs: netlink.LinkAttrs{
			Index: 12,
		},
		Table: 37,
	}
	peInterface := netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Index: 12,
		},
	}

	tests := []struct {
		name        string
		dst         string
		expectedDst string
	}{
		{
			name:        "/24 cidr",
			dst:         "192.168.10.3/24",
			expectedDst: "192.168.10.3/32",
		},
		{
			name:        "/28 cidr",
			dst:         "192.168.10.3/28",
			expectedDst: "192.168.10.3/32",
		},
	}
	for _, tc := range tests {
		route, err := hostIPToRoute(vrf, tc.dst, &peInterface)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		_, desiredDest, err := net.ParseCIDR(tc.expectedDst)
		if err != nil {
			t.Fatalf("failed to parse expected dst")
		}
		if desiredDest.String() != route.Dst.String() {
			t.Fatalf("expecting %s got %s", desiredDest, route.Dst)
		}
		if route.Table != int(vrf.Table) {
			t.Fatalf("expecting vrf table %d, got %d", vrf.Table, route.Table)
		}
		if route.LinkIndex != peInterface.Index {
			t.Fatalf("expecting pe interface index %d, got %d", peInterface.Index, route.LinkIndex)
		}
	}
}
