package frrconfig

import (
	"fmt"
	"log/slog"
	"os/exec"
)

type Action string

const (
	Test         = "test"
	Reload       = "reload"
	reloaderPath = "/usr/lib/frr/frr-reload.py"
)

const frrConfPath = "/etc/frr/frr.conf"

// Config reloads the frr configuration at the given path.
func Update(path string) error {
	err := reloadAction(path, Test)
	if err != nil {
		return err
	}
	err = reloadAction(path, Reload)
	if err != nil {
		return err
	}
	return nil
}

var execCommand = exec.Command

func reloadAction(path string, action Action) error {
	reloadParameter := "--" + string(action)
	cmd := execCommand("python3", "-c", reloaderPath, reloadParameter, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("frr update failed", "action", action, "error", err, "output", output)
		return fmt.Errorf("frr update %s failed: %w", action, err)
	}
	slog.Debug("frr update succeeded", "action", action, "output", output)
	return nil
}
