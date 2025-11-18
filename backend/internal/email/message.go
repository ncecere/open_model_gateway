package email

import "context"

// Message represents an email to be delivered to one or more recipients.
type Message struct {
	From     string
	To       []string
	Subject  string
	Body     string
	HTMLBody string
}

// Sender delivers email messages via a specific transport (SMTP, third-party API,
// etc).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
