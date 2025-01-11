package hostnetwork

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type VNIParams struct {
	VRF        string
	TargetNS   string
	VTEPIP     string
	VethHostIP string
	VethNSIP   string
	VNI        int
	VXLanPort  int
}

func SetupVNI(ctx context.Context, params VNIParams) error {
	slog.DebugContext(ctx, "setting up VNI", "params", params)
	defer slog.DebugContext(ctx, "end setting up VNI", "params", params)
	ns, err := netns.GetFromName(params.TargetNS)
	if err != nil {
		return fmt.Errorf("SetupVNI: Failed to get network namespace %s", params.TargetNS)
	}

	hostVeth, peVeth, err := setupVeth(ctx, params.VRF, ns)
	if err != nil {
		return err
	}
	err = assignIPToInterface(hostVeth, params.VethHostIP)
	if err != nil {
		return err
	}

	err = netlink.LinkSetUp(hostVeth)
	if err != nil {
		return fmt.Errorf("could not set link up for host leg %s: %v", hostVeth, err)
	}

	if err := inNamespace(ns, func() error {
		err = assignIPToInterface(peVeth, params.VethNSIP)
		if err != nil {
			return err
		}
		err = netlink.LinkSetUp(peVeth)
		if err != nil {
			return fmt.Errorf("could not set link up for host leg %s: %v", hostVeth, err)
		}

		slog.DebugContext(ctx, "setting up vrf", "vrf", params.VRF)
		vrf, err := setupVRF(params.VRF)
		if err != nil {
			return err
		}

		err = netlink.LinkSetMaster(peVeth, vrf)
		if err != nil {
			return fmt.Errorf("failed to set vrf %s as marter of pe veth %s", vrf.Name, peVeth.Attrs().Name)
		}

		slog.DebugContext(ctx, "setting up bridge")
		bridge, err := setupBridge(params, vrf)
		if err != nil {
			return err
		}

		slog.DebugContext(ctx, "setting up vxlan")
		err = setupVXLan(params, bridge)
		if err != nil {
			return err
		}

		slog.DebugContext(ctx, "setting up route to host")
		if err := addRouteToHost(vrf, params.VethHostIP, peVeth); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func RemoveNonConfiguredVNIs(ns netns.NsHandle, params []VNIParams) error {
	vrfs := map[string]bool{}
	vnis := map[int]bool{}
	for _, p := range params {
		vrfs[p.VRF] = true
		vnis[p.VNI] = true
	}
	hostLinks, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("remove non configured vnis: failed to list links: %w", err)
	}
	for _, hl := range hostLinks {
		if hl.Type() == "veth" && strings.HasPrefix(hl.Attrs().Name, HostVethPrefix) {
			vrf := vrfForHostLeg(hl.Attrs().Name)
			if vrfs[vrf] {
				continue
			}
			if err := netlink.LinkDel(hl); err != nil {
				return fmt.Errorf("remove host leg: %s %w", hl.Attrs().Name, err)
			}
		}
	}

	err = inNamespace(ns, func() error {
		links, err := netlink.LinkList()
		if err != nil {
			return fmt.Errorf("remove non configured vnis: failed to list links: %w", err)
		}

		if err := clearLinks("vxlan", vnis, links, vniFromVXLanName); err != nil {
			return err
		}
		if err := clearLinks("bridge", vnis, links, vniFromBridgeName); err != nil {
			return err
		}

		for _, l := range links {
			if l.Type() == "vrf" && !vrfs[l.Attrs().Name] {
				if err := netlink.LinkDel(l); err != nil {
					return fmt.Errorf("remove non configured vnis: failed to delete vrf %s %w", l.Attrs().Name, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func clearLinks(linkType string, vnis map[int]bool, links []netlink.Link, vniFromName func(string) (int, error)) error {
	for _, l := range links {
		if l.Type() == linkType {
			vni, err := vniFromName(l.Attrs().Name)
			if err != nil {
				return fmt.Errorf("remove non configured vnis: failed to get vni for %s %w", linkType, err)
			}
			if _, ok := vnis[vni]; ok {
				continue
			}
			if err := netlink.LinkDel(l); err != nil {
				return fmt.Errorf("remove non configured vnis: failed to delete %s %s %w", linkType, l.Attrs().Name, err)
			}
		}
	}

	return nil
}

func addRouteToHost(vrf *netlink.Vrf, dst string, peInterface netlink.Link) error {
	route, err := hostIPToRoute(vrf, dst, peInterface)
	if err != nil {
		return err
	}
	isPresent, err := checkRouteIsPresent(route)
	if err != nil {
		return err
	}
	if isPresent {
		return nil
	}

	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add route %v: %w", *route, err)
	}
	return nil
}

func hostIPToRoute(vrf *netlink.Vrf, dst string, peInterface netlink.Link) (*netlink.Route, error) {
	ip, dstCIDR, err := net.ParseCIDR(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to parse veth host ip %s: %w", dst, err)
	}
	_, maskSize := dstCIDR.Mask.Size()
	route := &netlink.Route{
		Table: int(vrf.Table),
		Dst: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(maskSize, maskSize),
		},
		LinkIndex: peInterface.Attrs().Index,
	}
	return route, nil

}

func checkRouteIsPresent(toCheck *netlink.Route) (bool, error) {
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
		Dst:   toCheck.Dst,
		Table: int(toCheck.Table),
	}, netlink.RT_FILTER_DST|netlink.RT_FILTER_TABLE)
	if err != nil {
		return false, fmt.Errorf("failed to list routes: %w", err)
	}

	for _, r := range routes {
		if !r.Dst.IP.Equal(toCheck.Dst.IP) {
			continue
		}
		if r.Dst.Mask.String() != toCheck.Dst.Mask.String() {
			continue
		}
		if r.LinkIndex != toCheck.LinkIndex {
			continue
		}
		if r.Table != toCheck.Table {
			continue
		}
		return true, nil
	}
	return false, nil
}
