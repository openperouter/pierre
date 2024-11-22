package frrconfig

import (
	"fmt"
	"net/http"
	"os"
)

func UpdaterForAddress(address string, configFile string) func(string) error {
	return func(config string) error {
		err := os.WriteFile(configFile, []byte(config), 0600)
		if err != nil {
			return fmt.Errorf("failed to write the config to %s", configFile)
		}
		requestURL := fmt.Sprintf("http://%s", address)
		res, err := http.Post(requestURL, "", nil)
		if err != nil {
			return fmt.Errorf("failed to reload against %s: %w", address, err)
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to reload against %s, status %d", address, res.StatusCode)
		}
		return nil
	}
}
