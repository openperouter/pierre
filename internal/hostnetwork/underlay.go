package hostnetwork

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	UnderlayLoopback = "lound"
	UnderlayNicAlias = "underlayNic"
)

// used to identify the interface moved into the network ns to serve
// the underlay
const underlayNicSpecialAddr = "172.16.1.1/32"

type UnderlayParams struct {
	MainNic  string
	VtepIP   string
	TargetNS string
}

func SetupUnderlay(ctx context.Context, params UnderlayParams) error {
	slog.DebugContext(ctx, "setup underlay", "params", params)
	defer slog.DebugContext(ctx, "setup underlay done")
	ns, err := netns.GetFromName(params.TargetNS)
	if err != nil {
		return fmt.Errorf("setupUnderlay: Failed to find network namespace %s: %w", params.TargetNS, err)
	}
	defer ns.Close()

	slog.DebugContext(ctx, "setup underlay", "step", "moving loopback interface")
	if err := inNamespace(ns, func() error {
		loopback, err := netlink.LinkByName(UnderlayLoopback)
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			slog.DebugContext(ctx, "setup underlay", "step", "creating loopback interface")
			loopback = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: UnderlayLoopback}}
			err = netlink.LinkAdd(loopback)
			if err != nil {
				return fmt.Errorf("assignVTEPToLoopback: failed to create loopback underlay")
			}
		}

		err = assignIPToInterface(loopback, params.VtepIP)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	err = moveUnderlayNic(ctx, params.MainNic, ns)
	if err != nil {
		return err
	}
	return nil
}

func moveUnderlayNic(ctx context.Context, underlayNic string, ns netns.NsHandle) error {
	oldUnderlayNic, err := oldUnderlayInterface(ns)
	if err != nil {
		return fmt.Errorf("failed to get old underlay interface %w", err)
	}

	if oldUnderlayNic != "" && oldUnderlayNic == underlayNic { // nothing to do
		slog.DebugContext(ctx, "move underlay", "event", "underlay nic already set")
		return nil
	}

	if oldUnderlayNic != "" && oldUnderlayNic != underlayNic { // need to move the old one back
		slog.DebugContext(ctx, "move underlay", "event", "different underlay nic found, removing", "old", oldUnderlayNic, "new", underlayNic)
		if err := removeUnderlayInterface(ctx, oldUnderlayNic, ns); err != nil {
			return err
		}
	}

	err = moveNicToNamespace(ctx, underlayNic, ns)
	if err != nil {
		return err
	}

	if err := inNamespace(ns, func() error {
		underlay, err := netlink.LinkByName(underlayNic)
		if err != nil {
			return fmt.Errorf("failed to get underlay nic by name %s: %w", underlayNic, err)
		}

		if err := assignIPToInterface(underlay, underlayNicSpecialAddr); err != nil {
			return err
		}
		if err := netlink.LinkSetUp(underlay); err != nil {
			return fmt.Errorf("could not set link up for VRF %s: %v", underlay.Attrs().Name, err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func removeUnderlayInterface(ctx context.Context, oldUnderlay string, ns netns.NsHandle) error {
	currentNS, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current ns: %w", err)
	}
	if err := inNamespace(ns, func() error {
		oldLink, err := netlink.LinkByName(oldUnderlay)
		if err != nil {
			return fmt.Errorf("failed to get old underlay by name %s under ns %s: %w", oldUnderlay, ns.String(), err)
		}
		addr, err := netlink.ParseAddr(underlayNicSpecialAddr)
		if err != nil {
			return fmt.Errorf("failed to parse special addr %s: %w", addr, err)
		}
		err = netlink.AddrDel(oldLink, addr)
		if err != nil {
			return fmt.Errorf("failed to remove special addr from %s %s: %w", oldLink.Attrs().Name, addr, err)
		}
		if err := moveNicToNamespace(ctx, oldLink.Attrs().Name, currentNS); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func oldUnderlayInterface(ns netns.NsHandle) (string, error) {
	res := ""
	err := inNamespace(ns, func() error {
		links, err := netlink.LinkList()
		if err != nil {
			return fmt.Errorf("failed to list links")
		}
		for _, l := range links {
			addr, _ := netlink.AddrList(l, netlink.FAMILY_ALL)
			slog.Debug("old underlay", "checking link", l.Attrs().Name, "addresses", addr)
			hasIP, err := interfaceHasIP(l, underlayNicSpecialAddr)
			if err != nil {
				return err
			}
			if hasIP {
				res = l.Attrs().Name
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if res != "" {
		slog.Debug("returning found has ip", "res", res)
		return res, nil
	}
	slog.Debug("returning not found")
	return "", nil
}
