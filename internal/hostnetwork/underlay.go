package hostnetwork

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const UnderlayLoopback = "lound"

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

	err = moveNicToNamespace(ctx, params.MainNic, ns)
	if err != nil {
		return err
	}

	inNamespace(ns, func() error {
		loopback, err := netlink.LinkByName(UnderlayLoopback)
		if errors.As(err, &netlink.LinkNotFoundError{}) {
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
	})
	return nil
}
