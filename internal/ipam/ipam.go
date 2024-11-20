package ipam

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

// IPsPerNode returns the ith and ith+1 ips from the given cidr, where i is the
// position of node in the provided list of nodes.
func SliceCIDR(pool string, index, size int) ([]net.IP, error) {
	_, ipNet, err := net.ParseCIDR(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cidr %s: %w", pool, err)
	}

	res := []net.IP{}
	for i := 0; i < size; i++ {
		ipIndex := size*index + i
		ip, err := cidr.Host(ipNet, ipIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to get %d address from %s: %w", ipIndex, pool, err)
		}
		res = append(res, ip)

	}

	return res, nil
}

func IPsInCIDR(pool string) (uint64, error) {
	_, ipNet, err := net.ParseCIDR(pool)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cidr %s: %w", pool, err)
	}

	return cidr.AddressCount(ipNet), nil
}

func NodeIndex(node string, nodes []string) (int, error) {
	for i, n := range nodes {
		if n == node {
			return i, nil
		}
	}
	return 0, fmt.Errorf("node %s not found in nodes %v", node, nodes)
}
