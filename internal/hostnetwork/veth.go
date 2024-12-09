package hostnetwork

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vishvananda/netlink"
)

func setupVeth(ctx context.Context, vrfName string) (netlink.Link, netlink.Link, error) {
	slog.DebugContext(ctx, "setting up veth", "veth", vrfName)
	hostSide, peSide := vethLegsForVRF(vrfName)

	link, err := netlink.LinkByName(hostSide)
	if err != nil && errors.As(err, &netlink.LinkNotFoundError{}) {
		link, err = createVeth(vrfName)
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
		vethHost, err = createVeth(vrfName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed create veth %s: %w", vrfName, err)
		}
	}
	if vethHost.PeerName != peSide {
		err := netlink.LinkDel(link)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete link %v: %w", link, err)
		}
		vethHost, err = createVeth(vrfName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed create veth %s: %w", vrfName, err)
		}
	}

	slog.DebugContext(ctx, "veth created veth", "veth", vrfName)
	peerIndex, err := netlink.VethPeerIndex(vethHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth for %s: %w", vrfName, err)
	}
	vethPE, err := netlink.LinkByIndex(peerIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find peer veth by index for %s: %w", vrfName, err)
	}
	slog.DebugContext(ctx, "veth is up", "vrf", vrfName)
	return vethHost, vethPE, nil
}

func createVeth(vrfName string) (*netlink.Veth, error) {
	hostSide, peSide := vethLegsForVRF(vrfName)
	vethHost := &netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: hostSide}, PeerName: peSide}
	err := netlink.LinkAdd(vethHost)
	if err != nil {
		return nil, fmt.Errorf("could not add veth %s: %w", vrfName, err)
	}
	return vethHost, nil
}

func vethLegsForVRF(name string) (string, string) {
	hostSide := HostVethPrefix + name
	peSide := "pe" + name
	return hostSide, peSide
}

const HostVethPrefix = "host"

func vrfForHostLeg(name string) string {
	return strings.TrimPrefix(name, HostVethPrefix)
}
