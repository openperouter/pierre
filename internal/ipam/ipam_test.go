package ipam

import "testing"

func TestIPam(t *testing.T) {
	nodes := []string{"nodea", "nodeb", "nodec", "noded"}
	tests := []struct {
		name        string
		cidr        string
		node        string
		expectedIP1 string
		expectedIP2 string
		shouldFail  bool
	}{
		{
			"simple",
			"192.168.1.0/24",
			"nodeb",
			"192.168.1.2",
			"192.168.1.3",
			false,
		},
		{
			"simplenodea",
			"192.168.1.0/24",
			"nodea",
			"192.168.1.0",
			"192.168.1.1",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip1, ip2, err := IPsPerNode(tc.node, nodes, tc.cidr)
			if err != nil && !tc.shouldFail {
				t.Fatalf("got error %s", err)
			}
			if err == nil && tc.shouldFail {
				t.Fatalf("expected error, did not happen")
			}
			if ip1.String() != tc.expectedIP1 {
				t.Fatalf("expecting %s got %s", tc.expectedIP1, ip1.String())
			}
			if ip2.String() != tc.expectedIP2 {
				t.Fatalf("expecting %s got %s", tc.expectedIP2, ip2.String())
			}

		})
	}
}
