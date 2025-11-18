package email

import "testing"

func TestRenderBudgetTemplate(t *testing.T) {
	tmpl, err := RenderBudgetAlertTemplate(BudgetAlertTemplateData{
		TenantName:       "Test Tenant",
		LevelLabel:       "Warning",
		LevelClass:       "warning",
		Timestamp:        "2025-01-01",
		CurrentSpend:     "$10",
		BudgetLimit:      "$100",
		WarningThreshold: "80%",
		BudgetReset:      "2025-02-01",
		ManageLink:       "https://example.com",
	})
	if err != nil {
		t.Fatalf("render err: %v", err)
	}
	if len(tmpl) == 0 {
		t.Fatalf("empty template")
	}
	//t.Log(tmpl)
}
