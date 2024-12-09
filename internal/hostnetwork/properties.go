package hostnetwork

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

func assignIPToInterface(link netlink.Link, address string) error {
	hasIP, err := interfaceHasIP(link, address)
	if err != nil {
		return err
	}
	if hasIP {
		return nil
	}
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("assignIPToInterface: failed to parse address %s for interface %s", address, link.Attrs().Name)
	}
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return fmt.Errorf("assignIPToInterface: failed to add address %s to interface %s, err %v", address, link.Attrs().Name, err)
	}
	return nil
}

func interfaceHasIP(link netlink.Link, address string) (bool, error) {
	_, err := netlink.ParseAddr(address)
	if err != nil {
		return false, fmt.Errorf("assignIPToInterface: failed to parse address %s for interface %s", address, link.Attrs().Name)
	}
	addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, fmt.Errorf("assignIPToInterface: failed to list addresses for interface %s", link.Attrs().Name)
	}
	for _, a := range addresses {
		if a.IPNet.String() == address {
			return true, nil
		}
	}
	return false, nil
}

func addrGenModeNone(l netlink.Link) error {
	fileName := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/addr_gen_mode", l.Attrs().Name)
	file, err := os.OpenFile(fileName, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("addrGenModeNone: error opening file: %w", err)
	}
	defer file.Close()
	if _, err := fmt.Fprintf(file, "%s\n", "1"); err != nil {
		return fmt.Errorf("addrGenModeNone: error writing to file: %w", err)
	}
	return nil
}

func setNeighSuppression(link netlink.Link) error {
	req := nl.NewNetlinkRequest(unix.RTM_SETLINK, unix.NLM_F_ACK)

	msg := nl.NewIfInfomsg(unix.AF_BRIDGE)
	msg.Index = int32(link.Attrs().Index)
	req.AddData(msg)

	br := nl.NewRtAttr(unix.IFLA_PROTINFO|unix.NLA_F_NESTED, nil)
	br.AddRtAttr(32, []byte{1})
	req.AddData(br)
	_, err := req.Execute(unix.NETLINK_ROUTE, 0)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	return nil
}

func moveNicToNamespace(ctx context.Context, nic string, ns netns.NsHandle) error {
	slog.DebugContext(ctx, "move nic to namespace", "nic", nic, "namespace", ns.String())
	defer slog.DebugContext(ctx, "move nic to namespace end", "nic", nic, "namespace", ns.String())

	err := inNamespace(ns, func() error {
		_, err := netlink.LinkByName(nic)
		if err != nil {
			return fmt.Errorf("setupUnderlay: Failed to find link %s: %w", nic, err)
		}
		return nil
	})
	if err == nil {
		slog.DebugContext(ctx, "nic is already in namespace", "nic", nic, "namespace", ns.String())
		return nil
	}

	link, err := netlink.LinkByName(nic)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to find link %s: %w", nic, err)
	}

	addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to get addresses for nic %s: %w", link.Attrs().Name, err)
	}

	err = netlink.LinkSetNsFd(link, int(ns))
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to move %s to network namespace %s: %w", link.Attrs().Name, ns.String(), err)
	}
	inNamespace(ns, func() error {
		err := netlink.LinkSetUp(link)
		if err != nil {
			return fmt.Errorf("setupUnderlay: Failed to set %s up in network namespace %s: %w", link.Attrs().Name, ns.String(), err)
		}

		for _, a := range addresses {
			err := netlink.AddrAdd(link, &a)
			if err != nil {
				return fmt.Errorf("moveNicToNamespace: Failed to add address %s to %s", a, link)
			}
		}
		return nil
	})
	return nil
}

func nsHasNic(nic, ns string) (bool, error) {
	_, err := netlink.LinkByName(nic)
	if err != nil && errors.As(err, &netlink.LinkNotFoundError{}) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to find link %s in ns %s: %w", nic, ns, err)
	}
	return true, nil
}
