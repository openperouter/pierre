package hostnetwork

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

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
		loopback, err := netlink.LinkByName("lo")
		if err != nil {
			return fmt.Errorf("assignVTEPToLoopback: failed to get lo")
		}
		err = assignIPToInterface(loopback, params.VtepIP)
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}
