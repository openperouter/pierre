package hostnetwork

import (
	"errors"
	"fmt"

	"github.com/vishvananda/netlink"
)

func setupVeth(name string) (netlink.Link, netlink.Link, error) {
	hostSide, peSide := namesForVeth(name)

	link, err := netlink.LinkByName(hostSide)
	if err != nil && errors.As(err, &netlink.LinkNotFoundError{}) {
		link, err = createVeth(name)
		if err != nil {
			return nil, nil, err
		}
	}

	vethHost, ok := link.(*netlink.Veth)
	if !ok {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
		vethHost, err = createVeth(name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed create veth %s: %w", name, err)
		}
	}
	if vethHost.PeerName != peSide {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
		vethHost, err = createVeth(name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed create veth %s: %w", name, err)
		}
	}

	err = netlink.LinkSetUp(vethHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not set veth %s up: %w", name, err)
	}
	peerIndex, err := netlink.VethPeerIndex(vethHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth for %s: %w", name, err)
	}
	vethPE, err := netlink.LinkByIndex(peerIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth by index for %s: %w", name, err)
	}
	return vethHost, vethPE, nil
}

func createVeth(name string) (*netlink.Veth, error) {
	hostSide, peSide := namesForVeth(name)
	vethHost := &netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: hostSide}, PeerName: peSide}
	err := netlink.LinkAdd(vethHost)
	if err != nil {
		return nil, fmt.Errorf("could not add veth %s: %w", name, err)
	}
	return vethHost, nil
}

func namesForVeth(name string) (string, string) {
	hostSide := name + "host"
	peSide := name + "pe"
	return hostSide, peSide
}
