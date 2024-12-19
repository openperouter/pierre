package ipam

import (
	"fmt"
	"testing"
)

func TestIPam(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		index       int
		expectedIP1 string
		expectedIP2 string
		shouldFail  bool
	}{
		{
			"first",
			"192.168.1.0/24",
			0,
			"192.168.1.0/31",
			"192.168.1.1/31",
			false,
		},
		{
			"second",
			"192.168.1.0/24",
			1,
			"192.168.1.2/31",
			"192.168.1.3/31",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ips, err := sliceCIDR(tc.cidr, tc.index, 2)
			if err != nil && !tc.shouldFail {
				t.Fatalf("got error %s", err)
			}
			if err == nil && tc.shouldFail {
				t.Fatalf("expected error, did not happen")
			}
			if len(ips) != 2 {
				t.Fatalf("expecting 2 ips, got %v", ips)
			}
			ip1, ip2 := ips[0], ips[1]
			if ip1.String() != tc.expectedIP1 {
				t.Fatalf("expecting %s got %s", tc.expectedIP1, ip1.String())
			}
			if ip2.String() != tc.expectedIP2 {
				t.Fatalf("expecting %s got %s", tc.expectedIP2, ip2.String())
			}

		})
	}
}

func TestVethIPs(t *testing.T) {
	tests := []struct {
		name         string
		pool         string
		index        int
		expectedHost string
		expectedPE   string
		shouldFail   bool
	}{
		{
			"first",
			"192.168.1.0/24",
			0,
			"192.168.1.0/32",
			"192.168.1.1/32",
			false,
		},
	}
	fmt.Println(tests)

}
