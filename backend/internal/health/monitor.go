package health

import (
	"context"
	"sync"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
)

// Monitor periodically pings provider routes and updates the router engine health state.
type Monitor struct {
	engine    *router.Engine
	interval  time.Duration
	timeout   time.Duration
	getRoutes func() map[string][]providers.Route
	startOnce sync.Once
}

// NewMonitor constructs a monitor using the health configuration.
func NewMonitor(engine *router.Engine, cfg config.HealthConfig) *Monitor {
	interval := cfg.CheckInterval
	if interval <= 0 {
		interval = time.Minute
	}
	timeout := cfg.Cooldown
	if timeout <= 0 || timeout > interval {
		timeout = 5 * time.Second
	}

	return &Monitor{
		engine:   engine,
		interval: interval,
		timeout:  timeout,
	}
}

// Start begins the monitoring loop until ctx is canceled.
func (m *Monitor) Start(ctx context.Context, getRoutes func() map[string][]providers.Route) {
	if getRoutes == nil || m.engine == nil {
		return
	}
	m.getRoutes = getRoutes

	m.startOnce.Do(func() {
		go m.run(ctx)
	})
}

func (m *Monitor) run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Initial sweep
	m.checkRoutes(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkRoutes(ctx)
		}
	}
}

func (m *Monitor) checkRoutes(ctx context.Context) {
	routes := m.getRoutes()
	if len(routes) == 0 {
		return
	}

	var wg sync.WaitGroup
	for alias, rs := range routes {
		for _, route := range rs {
			if route.Health == nil {
				continue
			}

			wg.Add(1)
			go func(alias string, route providers.Route) {
				defer wg.Done()
				timeoutCtx, cancel := context.WithTimeout(ctx, m.timeout)
				defer cancel()

				if err := route.Health(timeoutCtx); err != nil {
					m.engine.ReportFailure(alias, route)
					return
				}
				m.engine.ReportSuccess(alias, route)
			}(alias, route)
		}
	}
	wg.Wait()
}
