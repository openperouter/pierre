// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"text/template"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/openperouter/openperouter/internal/ipfamily"
)

var (
	configFileName      = "/etc/frr_reloader/frr.conf"
	reloaderPidFileName = "/etc/frr_reloader/reloader.pid"
	//go:embed templates/* templates/*
	templates embed.FS
)

type Config struct {
	Loglevel string
	Hostname string
	Underlay UnderlayConfig
	VNIs     []VNIConfig
}

type reloadEvent struct {
	config *Config
	useOld bool
}

type UnderlayConfig struct {
	MyASN     uint32
	VTEP      string
	Neighbors []NeighborConfig
}

type VNIConfig struct {
	ASN           uint32
	LocalNeighbor *NeighborConfig
	VRF           string
	VNI           int
}

type BFDProfile struct {
	Name             string
	ReceiveInterval  *uint32
	TransmitInterval *uint32
	DetectMultiplier *uint32
	EchoInterval     *uint32
	EchoMode         bool
	PassiveMode      bool
	MinimumTTL       *uint32
}

type NeighborConfig struct {
	Name          string
	ASN           uint32
	Addr          string
	Port          *uint16
	HoldTime      *uint64
	KeepaliveTime *uint64
	ConnectTime   *uint64
	Password      string
	BFDProfile    string
	EBGPMultiHop  bool
	IPFamily      ipfamily.Family
}

func (n *NeighborConfig) ID() string {
	return fmt.Sprintf("%s", n.Addr)
}

// templateConfig uses the template library to template
// 'globalConfigTemplate' using 'data'.
func templateConfig(data interface{}) (string, error) {
	counterMap := map[string]int{}
	t, err := template.New("frr.tmpl").Funcs(
		template.FuncMap{
			"counter": func(counterName string) int {
				counter := counterMap[counterName]
				counter++
				counterMap[counterName] = counter
				return counter
			},
			"dict": func(values ...interface{}) (map[string]interface{}, error) {
				if len(values)%2 != 0 {
					return nil, errors.New("invalid dict call, expecting even number of args")
				}
				dict := make(map[string]interface{}, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, fmt.Errorf("dict keys must be strings, got %v %T", values[i], values[i])
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
			"mustDisableConnectedCheck": func(ipFamily ipfamily.Family, myASN, asn uint32, eBGPMultiHop bool) bool {
				// return true only for IPv6 eBGP sessions
				if ipFamily == "ipv6" && myASN != asn && !eBGPMultiHop {
					return true
				}
				return false
			},
			"activateNeighborFor": func(ipFamily string, neighbourFamily ipfamily.Family) bool {
				return string(neighbourFamily) == ipFamily
			},
		}).ParseFS(templates, "templates/*")
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, data)
	return b.String(), err
}

// generateAndReloadConfigFile takes a 'struct Config' and, using a template,
// generates and writes a valid FRR configuration file. If this completes
// successfully it will also force FRR to reload that configuration file.
func generateAndReloadConfigFile(ctx context.Context, config *Config, updater ConfigUpdater) error {
	slog.InfoContext(ctx, "frr generate config", "event", "start")
	defer slog.InfoContext(ctx, "frr generate config", "event", "stop")

	slog.DebugContext(ctx, "frr generate config", "config", *config)

	configString, err := templateConfig(config)
	if err != nil {
		slog.Error("failed to generate config from template", "error", err, "cause", "template", "config", config)
		return err
	}
	err = updater(ctx, configString)
	if err != nil {
		slog.Error("failed to write frr config", "error", err, "cause", "updater", "config", config)
		return err
	}
	return nil
}

// debouncer takes a function that processes an Config, a channel where
// the update requests are sent, and squashes any requests coming in a given timeframe
// as a single request.
func debouncer(ctx context.Context, body func(config *Config) error,
	reload <-chan reloadEvent,
	reloadInterval time.Duration,
	failureRetryInterval time.Duration,
	l log.Logger) {
	go func() {
		var config *Config
		var timeOut <-chan time.Time
		timerSet := false
		for {
			select {
			case newCfg, ok := <-reload:
				if !ok { // the channel was closed
					return
				}
				if newCfg.useOld && config == nil {
					level.Debug(l).Log("op", "reload", "action", "ignore config", "reason", "nil config")
					continue // just ignore the event
				}
				if !newCfg.useOld && reflect.DeepEqual(newCfg.config, config) {
					level.Debug(l).Log("op", "reload", "action", "ignore config", "reason", "same config")
					continue // config hasn't changed
				}
				if !newCfg.useOld {
					config = newCfg.config
				}
				if !timerSet {
					timeOut = time.After(reloadInterval)
					timerSet = true
				}
			case <-timeOut:
				err := body(config)
				if err != nil {
					timeOut = time.After(failureRetryInterval)
					timerSet = true
					continue
				}
				timerSet = false
			case <-ctx.Done():
				return
			}
		}
	}()
}
