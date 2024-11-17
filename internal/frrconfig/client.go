package frrconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func SendUpdate(config, dest string) error {
	url := fmt.Sprintf("http://%s", dest)
	evt := Event{
		Config: config,
	}
	payloadBuf := new(bytes.Buffer)

	err := json.NewEncoder(payloadBuf).Encode(evt)
	if err != nil {
		return fmt.Errorf("failed to encode event : %w", err)
	}

	resp, err := http.Post(url, "text/plain", payloadBuf)
	if err != nil {
		return fmt.Errorf("failed to send config to %s: %w", url, err)
	}
	defer resp.Body.Close()
	return nil
}
