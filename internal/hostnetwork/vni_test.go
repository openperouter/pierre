package hostnetwork

import (
	"fmt"
	"os"
	"testing"

	"github.com/vishvananda/netlink"
)

const testNSName = "testns"

func TestVNI(t *testing.T) {
	cleanTest(t, testNSName)
	t.Cleanup(func() {
		cleanTest(t, testNSName)
	})
	_, testNS := createTestNS(t, testNSName)

	vtepIP := "192.170.0.9/24"
	vethHostIP := "192.168.9.0/32"
	vethNSIP := "192.168.9.0/32"
	vni := 100
	vxLanPort := 4789
	params := vniParams{
		Name:       "testvni",
		TargetNS:   testNSName,
		VTEPIP:     vtepIP,
		VethHostIP: vethHostIP,
		VethNSIP:   vethNSIP,
		VNI:        vni,
		VXLanPort:  vxLanPort,
	}
	err := SetupVNI(params)
	if err != nil {
		t.Fatalf("failed to setup vni: %v", err)
	}

	hostSide, _ := namesForVeth(params.Name)
	hostLegLink, err := netlink.LinkByName(hostSide)
	if err != nil {
		t.Fatalf("failed to get link by name: %v", err)
	}
	if hostLegLink.Attrs().OperState != netlink.OperUp {
		t.Fatalf("host leg is not up: %s", hostLegLink.Attrs().OperState)
	}
	hasIP, err := interfaceHasIP(hostLegLink, params.VethHostIP)
	if err != nil {
		t.Fatalf("failed to undersand if host leg has ip: %v", err)
	}
	if !hasIP {
		t.Fatalf("host leg doesn't have ip %s", params.VethHostIP)
	}
	_ = inNamespace(testNS, func() error {
		validateVNI(t, params)
		return nil
	})
}

func validateVNI(t *testing.T, params vniParams) {
	t.Helper()
	loopback, err := netlink.LinkByName("lo")
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

	bridgeLink, err := netlink.LinkByName(bridgeName(params.VNI))
	if err != nil {
		t.Fatalf("failed to get vxlan by name: %v", err)
	}
	bridge := bridgeLink.(*netlink.Bridge)
	if bridge.OperState != netlink.OperUp {
		t.Fatalf("bridge is not up: %s", bridge.OperState)
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

	_, peSide := namesForVeth(params.Name)
	peLegLink, err := netlink.LinkByName(peSide)
	if err != nil {
		t.Fatalf("failed to get peLegLink by name: %v", err)
	}
	if peLegLink.Attrs().OperState != netlink.OperUp {
		t.Fatalf("peLegLink is not up: %s", peLegLink.Attrs().OperState)
	}
	hasIP, err := interfaceHasIP(peLegLink, params.VethNSIP)
	if err != nil {
		t.Fatalf("failed to undersand if pe leg has ip: %v", err)
	}
	if !hasIP {
		t.Fatalf("pe leg doesn't have ip %s", params.VethNSIP)
	}
}

func checkAddrGenModeNone(t *testing.T, l netlink.Link) (bool, error) {
	t.Helper()
	fileName := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/addr_gen_mode", l.Attrs().Name)
	addrGenMode, err := os.ReadFile(fileName)
	if err != nil {
		return false, err
	}
	fmt.Printf("FEDE addr [%s]", string(addrGenMode))
	if string(addrGenMode) == "1" {
		fmt.Println("true")
		return true, nil
	}
	return false, nil
}
