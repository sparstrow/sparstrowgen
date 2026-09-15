package main

import (
	"context"
	"reflect"
	"time"

	"github.com/sparstrow/sparstrowgen/server/internal/agent"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
)

// providerCheckInterval is how often a connected computer looks for an agent
// CLI installed, removed or updated since it connected. Before this, the app
// learned about one only at the next connection.
var providerCheckInterval = envDuration("PROVIDER_CHECK_INTERVAL", 5*time.Minute)

// hello tells the server who this computer is and what it can run. It is sent
// on connecting, and again whenever the installed agents change; the server
// replaces the provider list each time.
func (d *daemon) hello(providers []protocol.Provider) error {
	return d.send(protocol.DaemonMessage{
		Type: protocol.DaemonHello, Machine: hostname(), Providers: providers,
		Version: version, Protocol: protocol.DaemonProtocol, SelfUpdates: d.updates != nil,
	})
}

// providerWatch decides when the installed agents need reporting again.
type providerWatch struct {
	fingerprint func() string
	detect      func(context.Context) []protocol.Provider
	last        string
	providers   []protocol.Provider
}

// check runs the full detection only when the cheap fingerprint changed, and
// returns the result only when it differs from what was last reported: a CLI
// update that changes nothing the app shows is not worth a message.
func (w *providerWatch) check(ctx context.Context) ([]protocol.Provider, bool) {
	fp := w.fingerprint()
	if fp == w.last {
		return nil, false
	}
	w.last = fp
	next := w.detect(ctx)
	if reflect.DeepEqual(next, w.providers) {
		return nil, false
	}
	w.providers = next
	return next, true
}

func (d *daemon) watchProviders(ctx context.Context, done <-chan struct{}, detect func(context.Context) []protocol.Provider, providers []protocol.Provider) {
	w := &providerWatch{fingerprint: agent.Fingerprint, detect: detect, last: agent.Fingerprint(), providers: providers}
	tick := time.NewTicker(providerCheckInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-tick.C:
			next, changed := w.check(ctx)
			if !changed {
				continue
			}
			for _, p := range next {
				d.log.Info("provider changed", "id", p.ID, "availability", p.Availability, "models", len(p.Models))
			}
			if err := d.hello(next); err != nil {
				d.log.Warn("could not report the changed agents", "err", err)
			}
		}
	}
}
