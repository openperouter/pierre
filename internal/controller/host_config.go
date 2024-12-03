package controller

import (
	"context"
	"fmt"

	"github.com/openperouter/openperouter/api/v1alpha1"
	"github.com/openperouter/openperouter/internal/conversion"
	"github.com/openperouter/openperouter/internal/hostnetwork"
	"github.com/openperouter/openperouter/internal/pods"
)

type interfacesConfiguration struct {
	RouterPodUUID string `json:"routerPodUUID,omitempty"`
	PodRuntime    pods.Runtime
	NodeIndex     int                 `json:"nodeIndex,omitempty"`
	Underlays     []v1alpha1.Underlay `json:"underlays,omitempty"`
	Vnis          []v1alpha1.VNI      `json:"vnis,omitempty"`
}

func configureInterfaces(ctx context.Context, config interfacesConfiguration) error {
	targetNS, err := config.PodRuntime.NetworkNamespace(ctx, config.RouterPodUUID)
	if err != nil {
		return fmt.Errorf("failed to retrieve namespace for pod %s: %w", config.RouterPodUUID, err)
	}
	// TODO log
	underlayParams, vnis, err := conversion.APItoHostConfig(config.NodeIndex, targetNS, config.Underlays, config.Vnis)
	if err != nil {
		return fmt.Errorf("failed to convert config to host configuration: %w", err)
	}
	if err := hostnetwork.SetupUnderlay(underlayParams); err != nil {
		return fmt.Errorf("failed to setup underlay: %w", err)
	}
	for _, vni := range vnis {
		// TODO log
		if err := hostnetwork.SetupVNI(vni); err != nil {
			return fmt.Errorf("failed to setup vni: %w", err)
		}
	}
	return nil
}
