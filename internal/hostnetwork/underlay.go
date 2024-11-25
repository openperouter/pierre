package hostnetwork

type UnderlayParams struct {
	mainNic string
	vtepIP  string
}

func setupUnderlay(params UnderlayParams, targetNS string) error {
	return nil
}
