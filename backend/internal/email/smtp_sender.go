package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// SMTPSender delivers emails using the configured SMTP server.
type SMTPSender struct {
	cfg config.SMTPConfig
}

// NewSMTPSender constructs a Sender backed by SMTP. When the host/from fields
// are empty the function returns nil so callers can easily detect the absence of
// a mail transport.
func NewSMTPSender(cfg config.SMTPConfig) Sender {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port == 0 || strings.TrimSpace(cfg.From) == "" {
		return nil
	}
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
		cfg.ConnectTimeout = timeout
	}
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if s == nil {
		return fmt.Errorf("smtp sender not configured")
	}
	recipients := normalizeAddresses(msg.To)
	if len(recipients) == 0 {
		return nil
	}
	wireMsg := buildRFC822Message(Message{
		From:     s.cfg.From,
		To:       recipients,
		Subject:  msg.Subject,
		Body:     msg.Body,
		HTMLBody: msg.HTMLBody,
	})
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
		if err := client.Rcpt(rcpt); err != nil {
			client.Quit()
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		client.Quit()
		return err
	}
	if _, err := writer.Write(wireMsg); err != nil {
		_ = writer.Close()
		client.Quit()
		return err
	}
	if err := writer.Close(); err != nil {
		client.Quit()
		return err
	}
	return client.Quit()
}

func (s *SMTPSender) newClient(ctx context.Context, addr string) (*smtp.Client, error) {
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
		tlsCfg := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: s.cfg.SkipTLSVerify,
		}
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

func buildRFC822Message(msg Message) []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(normalizeAddresses(msg.To), ",")))
	if strings.TrimSpace(msg.Subject) != "" {
		buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	if strings.TrimSpace(msg.HTMLBody) != "" {
		boundary := fmt.Sprintf("omg-%d", time.Now().UnixNano())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		if strings.TrimSpace(msg.Body) != "" {
			buf.WriteString(normalizeNewlines(msg.Body))
		}
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		return buf.Bytes()
	}
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(normalizeNewlines(msg.Body))
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func normalizeAddresses(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func normalizeNewlines(body string) string {
	return strings.ReplaceAll(body, "\r\n", "\n")
}
