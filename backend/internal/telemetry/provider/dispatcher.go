package provider

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// AlertConfig captures delivery channels/cooldown for provider incidents.
type AlertConfig struct {
	Emails   []string
	Webhooks []string
	Cooldown time.Duration
}

// IncidentDispatcher handles cooldown/dedupe for provider alerts.
type IncidentDispatcher struct {
	sink    AlertSink
	cfg     AlertConfig
	state   map[string]alertState
	stateMu sync.Mutex
}

type alertState struct {
	sentAt time.Time
}

func NewIncidentDispatcher(cfg AlertConfig, sink AlertSink) *IncidentDispatcher {
	if sink == nil {
		sink = NoopAlertSink{}
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = time.Hour
	}
	return &IncidentDispatcher{
		sink:  sink,
		cfg:   cfg,
		state: make(map[string]alertState),
	}
}

func (d *IncidentDispatcher) Dispatch(ctx context.Context, inc Incident) error {
	key := incidentKey(inc)
	if !d.shouldSend(key, inc.OpenedAt) {
		return nil
	}
	if err := d.sink.Notify(ctx, inc); err != nil {
		return err
	}
	d.stateMu.Lock()
	d.state[key] = alertState{sentAt: time.Now().UTC()}
	d.stateMu.Unlock()
	return nil
}

func (d *IncidentDispatcher) shouldSend(key string, ts time.Time) bool {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	state, ok := d.state[key]
	if !ok {
		return true
	}
	return ts.Sub(state.sentAt) >= d.cfg.Cooldown
}

func incidentKey(inc Incident) string {
	return strings.Join([]string{inc.Provider, inc.ModelAlias, inc.Type}, "|")
}

// ProviderAlertConfig builds AlertConfig from runtime config.
func ProviderAlertConfig(cfg config.TelemetryConfig, budget config.BudgetConfig) AlertConfig {
	cooldown := budget.Alert.Cooldown
	if cooldown <= 0 {
		cooldown = time.Hour
	}
	return AlertConfig{
		Emails:   budget.Alert.Emails,
		Webhooks: budget.Alert.Webhooks,
		Cooldown: cooldown,
	}
}
