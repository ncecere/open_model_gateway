package user

import (
	"testing"

	decimal "github.com/shopspring/decimal"
)

func TestBuildMemberBudget(t *testing.T) {
	if got := buildMemberBudget(decimal.Zero, decimal.Zero, 0); got != nil {
		t.Fatalf("expected nil budget for zero values, got %+v", got)
	}

	budget := buildMemberBudget(decimal.NewFromFloat(25.5), decimal.NewFromFloat(0.75), 1000)
	if budget == nil {
		t.Fatal("expected budget payload")
	}
	if budget.BudgetUSD != 25.5 {
		t.Fatalf("expected budget_usd 25.5, got %v", budget.BudgetUSD)
	}
	if budget.WarningThreshold != 0.75 {
		t.Fatalf("expected warning_threshold 0.75, got %v", budget.WarningThreshold)
	}
	if budget.TokenCap != 1000 {
		t.Fatalf("expected token_cap 1000, got %d", budget.TokenCap)
	}
}
