package ipam

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

type Veths struct {
	HostSide      net.IPNet
	ContainerSide net.IPNet
}

func VethIPs(pool string, index int) (Veths, error) {
	_, cidr, err := net.ParseCIDR(pool)
	if err != nil {
		return Veths{}, fmt.Errorf("failed to parse pool %s: %w", pool, err)
	}
	peSide, err := cidrElem(pool, 32, 0) // pe side is always the same
	if err != nil {
		return Veths{}, err
	}
	ones, _ := cidr.Mask.Size()
	hostSide, err := cidrElem(pool, ones, index+1)
	if err != nil {
		return Veths{}, err
	}
	return Veths{HostSide: *hostSide, ContainerSide: *peSide}, nil
}

func VETPIp(pool string, index int) (string, error) {
	ips, err := sliceCIDR(pool, index, 1)
	if err != nil {
		return "", err
	}
	if len(ips) != 1 {
		return "", fmt.Errorf("vtepIP, expecting 1 ip, got %v", ips)
	}
	return ips[0].IP.String() + "/32", nil
}

// cidrElem returns the ith elem of len size for the given cidr.
func cidrElem(pool string, mask int, index int) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cidr %s: %w", pool, err)
	}

	ip, err := cidr.Host(ipNet, index)
	if err != nil {
		return nil, fmt.Errorf("failed to get %d address from %s: %w", index, pool, err)
	}
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(mask, 32),
	}, nil
}

// sliceCIDR returns the ith block of len size for the given cidr.
func sliceCIDR(pool string, index, size int) ([]net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cidr %s: %w", pool, err)
	}

	res := []net.IPNet{}
	for i := 0; i < size; i++ {
		ipIndex := size*index + i
		ip, err := cidr.Host(ipNet, ipIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to get %d address from %s: %w", ipIndex, pool, err)
		}
		ipNet := net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(31, 32),
		}

		res = append(res, ipNet)
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
