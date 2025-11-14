package router

import (
	"testing"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

func TestEngineSelectRoutesSkipsOpenCircuits(t *testing.T) {
	engine := NewEngine()
	alias := "gpt-test"
	healthy := providers.Route{Alias: alias, Model: "m1", Metadata: map[string]string{"deployment": "m1"}, Weight: 1}
	unhealthy := providers.Route{Alias: alias, Model: "m2", Metadata: map[string]string{"deployment": "m2"}, Weight: 1}

	engine.routes[alias] = []providers.Route{healthy, unhealthy}
	engine.state[routeKey(alias, healthy)] = &routeState{}
	engine.state[routeKey(alias, unhealthy)] = &routeState{openUntil: time.Now().Add(time.Minute)}

	selected := engine.SelectRoutes(alias)
	if len(selected) != 1 {
		t.Fatalf("expected 1 healthy route, got %d", len(selected))
	}
	if selected[0].Metadata["deployment"] != "m1" {
		t.Fatalf("expected healthy route first, got %v", selected[0])
	}
}

func TestEngineCircuitBreakerTransitions(t *testing.T) {
	engine := NewEngine()
	alias := "gpt-breaker"
	route := providers.Route{Alias: alias, Model: "m1", Metadata: map[string]string{"deployment": "m1"}}

	for i := 0; i < failureThreshold; i++ {
		engine.ReportFailure(alias, route)
	}
	st := engine.state[routeKey(alias, route)]
	if st == nil {
		t.Fatalf("route state missing")
	}
	if st.consecutiveFailures != failureThreshold {
		t.Fatalf("expected %d failures, got %d", failureThreshold, st.consecutiveFailures)
	}
	if !st.openUntil.After(time.Now()) {
		t.Fatalf("circuit should be open")
	}

	engine.ReportSuccess(alias, route)
	if st.consecutiveFailures != 0 {
		t.Fatalf("success should reset failures, got %d", st.consecutiveFailures)
	}
	if !st.openUntil.IsZero() {
		t.Fatalf("openUntil should reset, got %s", st.openUntil)
	}
}

func TestMergeEntriesPrioritizesSources(t *testing.T) {
	enabled := true
	cfgEntries := []config.ModelCatalogEntry{{
		Alias:         "gpt",
		Provider:      "cfg",
		ProviderModel: "cfg-model",
		Enabled:       &enabled,
		Metadata:      map[string]string{"from": "config"},
	}}

	dbEntries := []db.ModelCatalog{{
		Alias:         "db-only",
		Provider:      "db",
		ProviderModel: "db-model",
		Enabled:       false,
		Weight:        5,
	}}

	merged, err := MergeEntries(cfgEntries, dbEntries)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(merged))
	}

	flags := map[string]bool{}
	for _, entry := range merged {
		flags[entry.Alias] = true
		if entry.Alias == "db-only" && entry.Provider != "db" {
			t.Fatalf("db entry should override provider, got %s", entry.Provider)
		}
	}
	if !flags["gpt"] || !flags["db-only"] {
		t.Fatalf("merged aliases missing: %v", flags)
	}
}
