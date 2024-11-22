package ipam

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

type Veths struct {
	HostSide      string
	ContainerSide string
}

func VethIPs(pool string, index int) (Veths, error) {
	ips, err := sliceCIDR(pool, index, 2)
	if err != nil {
		return Veths{}, err
	}
	if len(ips) != 1 {
		return Veths{}, fmt.Errorf("veths, expecting 2 ip, got %v", ips)
	}
	return Veths{HostSide: ips[0].String(), ContainerSide: ips[1].String()}, nil
}

func VETPIp(pool string, index int) (string, error) {
	ips, err := sliceCIDR(pool, index, 1)
	if err != nil {
		return "", err
	}
	if len(ips) != 1 {
		return "", fmt.Errorf("vtepIP, expecting 1 ip, got %v", ips)
	}
	return ips[0].String(), nil
}

// sliceCIDR returns the ith block of len size for the given cidr.
func sliceCIDR(pool string, index, size int) ([]net.IP, error) {
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
