package controller

import (
	"fmt"
	"log/slog"

	"github.com/openperouter/openperouter/api/v1alpha1"
	"github.com/openperouter/openperouter/internal/conversion"
	"github.com/openperouter/openperouter/internal/frr"
	"github.com/openperouter/openperouter/internal/frrconfig"
)

type frrConfigData struct {
	configFile string
	address    string
	port       int
	nodeIndex  int
	underlays  []v1alpha1.Underlay
	vnis       []v1alpha1.VNI
}

func reloadFRRConfig(data frrConfigData) error {
	slog.Debug("reloading FRR config", "config", data)
	frrConfig, err := conversion.APItoFRR(data.nodeIndex, data.underlays, data.vnis)
	if err != nil {
		return fmt.Errorf("failed to generate the frr configuration: %w", err)
	}

	url := fmt.Sprintf("%s:%d", data.address, data.port)
	updater := frrconfig.UpdaterForAddress(url, data.configFile)
	err = frr.ApplyConfig(&frrConfig, updater)
	if err != nil {
		return fmt.Errorf("failed to update the frr configuration: %w", err)
	}
	return nil
}
