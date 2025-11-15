package usagepipeline

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// SMTPSink sends budget alerts via SMTP email.
type SMTPSink struct {
	cfg config.SMTPConfig
}

func NewSMTPSink(cfg config.SMTPConfig, _ *slog.Logger) AlertSink {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port == 0 || strings.TrimSpace(cfg.From) == "" {
		return nil
	}
	return &SMTPSink{cfg: cfg}
}

func (s *SMTPSink) Notify(ctx context.Context, payload AlertPayload) error {
	if s == nil {
		return nil
	}
	recipients := payload.Channels.Emails
	if len(recipients) == 0 {
		return nil
	}

	msg := buildEmailMessage(s.cfg.From, recipients, payload)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	client, err := s.newClient(ctx, addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Mail(s.cfg.From); err != nil {
		client.Quit()
		return err
	}
	for _, rcpt := range recipients {
		if strings.TrimSpace(rcpt) == "" {
			continue
		}
		if err := client.Rcpt(rcpt); err != nil {
			client.Quit()
			return err
		}
	}
	wc, err := client.Data()
	if err != nil {
		client.Quit()
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		client.Quit()
		return err
	}
	if err := wc.Close(); err != nil {
		client.Quit()
		return err
	}
	return client.Quit()
}

func (s *SMTPSink) newClient(ctx context.Context, addr string) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: s.cfg.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	host := s.cfg.Host
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if s.cfg.UseTLS {
		tlsCfg := &tls.Config{ServerName: host, InsecureSkipVerify: s.cfg.SkipTLSVerify}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				client.Close()
				return nil, err
			}
		}
	}

	if strings.TrimSpace(s.cfg.Username) != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			client.Close()
			return nil, err
		}
	}
	return client, nil
}

func buildEmailMessage(from string, to []string, payload AlertPayload) []byte {
	subject := fmt.Sprintf("[Budget %s] Tenant %s", strings.ToUpper(string(payload.Level)), payload.TenantID)
	if payload.Guardrail != nil {
		subject = fmt.Sprintf("[Guardrail] Tenant %s", payload.TenantID)
	}
	body := formatEmailBody(payload)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func formatEmailBody(payload AlertPayload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tenant ID: %s\n", payload.TenantID)
	if payload.Guardrail != nil {
		fmt.Fprintf(&b, "Type: Guardrail\n")
		fmt.Fprintf(&b, "Stage: %s\n", payload.Guardrail.Stage)
		fmt.Fprintf(&b, "Action: %s\n", payload.Guardrail.Action)
		if payload.Guardrail.Category != "" {
			fmt.Fprintf(&b, "Category: %s\n", payload.Guardrail.Category)
		}
		if len(payload.Guardrail.Violations) > 0 {
			fmt.Fprintf(&b, "Violations: %s\n", strings.Join(payload.Guardrail.Violations, ", "))
		}
		if payload.APIKeyPrefix != "" {
			fmt.Fprintf(&b, "API Key Prefix: %s\n", payload.APIKeyPrefix)
		}
		if payload.ModelAlias != "" {
			fmt.Fprintf(&b, "Model Alias: %s\n", payload.ModelAlias)
		}
		fmt.Fprintf(&b, "Timestamp: %s\n", payload.Timestamp.UTC().Format(time.RFC3339))
		return b.String()
	}
	limit := formatCurrency(payload.Status.LimitCents)
	spend := formatCurrency(payload.Status.TotalCostCents)
	fmt.Fprintf(&b, "Type: Budget\n")
	fmt.Fprintf(&b, "Level: %s\n", strings.ToUpper(string(payload.Level)))
	fmt.Fprintf(&b, "Spend: %s / %s\n", spend, limit)
	fmt.Fprintf(&b, "Exceeded: %t\n", payload.Status.Exceeded)
	fmt.Fprintf(&b, "Warning: %t\n", payload.Status.Warning)
	if payload.APIKeyPrefix != "" {
		fmt.Fprintf(&b, "API Key Prefix: %s\n", payload.APIKeyPrefix)
	}
	if payload.ModelAlias != "" {
		fmt.Fprintf(&b, "Model Alias: %s\n", payload.ModelAlias)
	}
	fmt.Fprintf(&b, "Timestamp: %s\n", payload.Timestamp.UTC().Format(time.RFC3339))
	return b.String()
}

func formatCurrency(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
