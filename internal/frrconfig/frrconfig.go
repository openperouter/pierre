package frrconfig

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Event struct {
	When   time.Time
	Config string
}

type Action string

const (
	Test         = "test"
	Reload       = "reload"
	reloaderPath = "/usr/lib/frr/frr-reload.py"
)

var frrConfPath = "/etc/frr/frr.conf.new"
var reloadMutex sync.Mutex

// Config reloads the frr configuration at the given path.
func Update(e Event) error {
	reloadMutex.Lock()
	defer reloadMutex.Unlock()

	if err := os.WriteFile(frrConfPath, []byte(e.Config), 0666); err != nil {
		return fmt.Errorf("failed to write the configuration file %w", err)
	}

	err := reloadAction(frrConfPath, Test)
	if err != nil {
		return err
	}
	err = reloadAction(frrConfPath, Reload)
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
