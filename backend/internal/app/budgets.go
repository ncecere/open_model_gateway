package app

import (
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// UpdateBudgetConfig swaps the in-memory config and notifies dependent services.
func (c *Container) UpdateBudgetConfig(cfg config.BudgetConfig) {
	c.Config.Budgets = cfg
	if c.UsageLogger != nil {
		c.UsageLogger.SetConfig(cfg)
	}
}
