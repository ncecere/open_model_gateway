package email

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

var (
	htmlTemplates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))
)

const (
	budgetAlertTemplateName = "budget_alert.html"
	providerAlertTemplateName = "provider_alert.html"
	adminInviteTemplateName = "admin_invite.html"
	testEmailTemplateName   = "test_email.html"
)

type BudgetAlertTemplateData struct {
	TenantName       string
	LevelLabel       string
	LevelClass       string
	Timestamp        string
	CurrentSpend     string
	BudgetLimit      string
	WarningThreshold string
	BudgetReset      string
	ManageLink       string
}

type ProviderAlertTemplateData struct {
	Provider         string
	Alias            string
	IncidentType     string
	IncidentTypeLabel string
	Timestamp        string
	RequestCount     string
	WindowLabel      string
	ObservedValue    string
	ThresholdValue   string
	SampleError      string
	ManageLink       string
}

type AdminInviteTemplateData struct {
	RecipientName string
	PortalURL     string
}

type TestEmailTemplateData struct {
	Recipient string
	Timestamp string
}

func renderTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := htmlTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderBudgetAlertTemplate(data BudgetAlertTemplateData) (string, error) {
	return renderTemplate(budgetAlertTemplateName, data)
}

func RenderProviderAlertTemplate(data ProviderAlertTemplateData) (string, error) {
	return renderTemplate(providerAlertTemplateName, data)
}

func RenderAdminInviteTemplate(data AdminInviteTemplateData) (string, error) {
	return renderTemplate(adminInviteTemplateName, data)
}

func RenderTestEmailTemplate(data TestEmailTemplateData) (string, error) {
	return renderTemplate(testEmailTemplateName, data)
}
