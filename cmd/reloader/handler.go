package main

import (
	"net/http"

	"github.com/openperouter/openperouter/internal/reload"
)

const frrPath = "/etc/frr/frr.conf"

var reloadConfig = reload.Config

func reloadHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusBadRequest)
		return
	}
	err := reloadConfig(frrPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
