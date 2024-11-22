// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"os"
	"sync"

	"github.com/go-kit/log"
)

type ConfigUpdater func(string) error

type FRR struct {
	reloadConfig chan reloadEvent
	configFile   string
	logLevel     string
	sync.Mutex
}

const ReloadSuccess = "success"

// Create a variable for os.Hostname() in order to make it easy to mock out
// in unit tests.
var osHostname = os.Hostname

func ApplyConfig(config *Config, updater ConfigUpdater) error {
	hostname, err := osHostname()
	if err != nil {
		return err
	}

	config.Hostname = hostname
	return generateAndReloadConfigFile(config, updater)
}

func NewFRR(logger log.Logger) *FRR {
	res := &FRR{}
	return res
}

/*
func logLevelToFRR(level logging.Level) string {
	// Allowed frr log levels are: emergencies, alerts, critical,
	// 		errors, warnings, notifications, informational, or debugging
	switch level {
	case logging.LevelAll, logging.LevelDebug:
		return "debugging"
	case logging.LevelInfo:
		return "informational"
	case logging.LevelWarn:
		return "warnings"
	case logging.LevelError:
		return "error"
	case logging.LevelNone:
		return "emergencies"
	}

	return "informational"
}
*/
