// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
)

type ConfigHandler interface {
	ApplyConfig(config *Config) error
}

type StatusFetcher interface {
	GetStatus() Status
}

type StatusChanged func()

type Status struct {
	updateTime       string
	Current          string
	LastReloadResult string
}

type FRR struct {
	reloadConfig chan reloadEvent
	logLevel     string
	Status       Status
	sync.Mutex
}

const ReloadSuccess = "success"

// Create a variable for os.Hostname() in order to make it easy to mock out
// in unit tests.
var osHostname = os.Hostname

func (f *FRR) ApplyConfig(config *Config) error {
	hostname, err := osHostname()
	if err != nil {
		return err
	}

	// TODO add internal wrapper
	config.Loglevel = f.logLevel
	config.Hostname = hostname
	f.reloadConfig <- reloadEvent{config: config}
	return nil
}

var debounceTimeout = 3 * time.Second
var failureTimeout = time.Second * 5

func NewFRR(ctx context.Context, configFile string, logger log.Logger) *FRR {
	res := &FRR{
		reloadConfig: make(chan reloadEvent),
		// logLevel:        logLevelToFRR(logLevel), TODO
		// onStatusChanged: onStatusChanged,
	}
	reload := func(config *Config) error {
		return generateAndReloadConfigFile(config, configFile, logger)
	}

	debouncer(ctx, reload, res.reloadConfig, debounceTimeout, failureTimeout, logger)
	return res
}

func (f *FRR) GetStatus() Status {
	f.Lock()
	defer f.Unlock()
	return f.Status
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
