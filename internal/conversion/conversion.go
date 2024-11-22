package conversion

import (
	"github.com/openperouter/openperouter/api/v1alpha1"
	"github.com/openperouter/openperouter/internal/frr"
)

func APItoFRR(underlays []v1alpha1.Underlay, vnis []v1alpha1.VNI) (frr.Config, error) {
	return frr.Config{}, nil
}
