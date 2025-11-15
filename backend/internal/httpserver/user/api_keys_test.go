package user

import "testing"

func TestValidateQuotaForLimit(t *testing.T) {
	if err := validateQuotaForLimit(&quotaPayload{BudgetUSD: 150}, 100); err == nil {
		t.Fatal("expected error when budget exceeds limit")
	}
	if err := validateQuotaForLimit(&quotaPayload{WarningThreshold: 1.5}, 200); err == nil {
		t.Fatal("expected threshold validation error")
	}
	if err := validateQuotaForLimit(&quotaPayload{BudgetUSD: 80, WarningThreshold: 0.8}, 100); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
