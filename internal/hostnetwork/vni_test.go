package hostnetwork

import (
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
	_ = inNamespace(testNS, func() error {
		// TODO assertions
		return nil
	})
}

func validateVNI(t *testing.T, params vniParams) {
	t.Helper()
	validateVXLan := func(t *testing.T, l netlink.Link, params vniParams) {

	}
	links, err := netlink.LinkList()
	if err != nil {
		t.Fatalf("failed to list links %v", err)
	}
	for _, l := range links {
		if l.Type() == "vxlan" {
			validateVXLan(t, l, params)
		}

	}
}
