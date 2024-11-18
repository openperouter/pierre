// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
)

const timer = 10 * time.Millisecond
const failureTimer = 10 * time.Millisecond

func TestDebounce(t *testing.T) {
	result := make(chan *Config, 10) // buffered to accommodate spurious rewrites
	dummyUpdate := func(config *Config) error {
		result <- config
		return nil
	}

	reload := make(chan reloadEvent)
	defer close(reload)
	debouncer(context.Background(), dummyUpdate, reload, timer, failureTimer, log.NewNopLogger())
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 1}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 2}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 3}}}
	if len(result) != 0 {
		t.Fatal("received update before time")
	}
	time.Sleep(3 * timer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated := <-result
	if updated.Underlay.MyASN != 3 {
		t.Fatal("Config was not updated")
	}

	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 3}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 4}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 5}}}
	time.Sleep(3 * timer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated = <-result
	if updated.Underlay.MyASN != 5 {
		t.Fatal("Config was not updated")
	}
}

func TestDebounceRetry(t *testing.T) {
	result := make(chan *Config, 10) // buffered to accommodate spurious rewrites
	count := 0
	dummyUpdate := func(config *Config) error {
		count++
		if count <= 3 {
			return fmt.Errorf("error")
		}
		result <- config
		return nil
	}

	reload := make(chan reloadEvent)
	defer close(reload)
	debouncer(context.Background(), dummyUpdate, reload, timer, failureTimer, log.NewNopLogger())

	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 1}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 2}}}
	if len(result) != 0 {
		t.Fatal("received update before time")
	}
	time.Sleep(10 * failureTimer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated := <-result
	if updated.Underlay.MyASN != 2 {
		t.Fatal("Config was not updated")
	}
}

func TestDebounceReuseOld(t *testing.T) {
	result := make(chan *Config, 10) // buffered to accommodate spurious rewrites
	dummyUpdate := func(config *Config) error {
		result <- config
		return nil
	}

	reload := make(chan reloadEvent)
	defer close(reload)
	debouncer(context.Background(), dummyUpdate, reload, timer, failureTimer, log.NewNopLogger())

	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 1}}}
	if len(result) != 0 {
		t.Fatal("received update before time")
	}
	time.Sleep(3 * timer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated := <-result
	if updated.Underlay.MyASN != 1 {
		t.Fatal("Config was not updated")
	}
	// reload to see if the debouncer uses the old config
	reload <- reloadEvent{useOld: true}
	time.Sleep(3 * timer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated = <-result
	if updated.Underlay.MyASN != 1 {
		t.Fatal("Config was not updated")
	}
}

func TestDebounceSameConfig(t *testing.T) {
	result := make(chan *Config, 10) // buffered to accommodate spurious rewrites
	dummyUpdate := func(config *Config) error {
		result <- config
		return nil
	}

	reload := make(chan reloadEvent)
	defer close(reload)
	debouncer(context.Background(), dummyUpdate, reload, timer, failureTimer, log.NewNopLogger())
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 1}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 2}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 3, VTEP: "foo"}}}
	if len(result) != 0 {
		t.Fatal("received update before time")
	}
	time.Sleep(3 * timer)
	if len(result) != 1 {
		t.Fatal("received extra updates", len(result))
	}
	updated := <-result
	if updated.Underlay.MyASN != 3 {
		t.Fatal("Config was not updated")
	}

	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 3, VTEP: "foo"}}}
	reload <- reloadEvent{config: &Config{Underlay: UnderlayConfig{MyASN: 3, VTEP: "foo"}}}

	time.Sleep(3 * timer)
	if len(result) != 0 {
		updated := <-result
		t.Fatalf("received extra updates: %d %s", len(result), updated.Underlay.VTEP)
	}
}
