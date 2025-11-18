package email

import "context"

// Message represents a plain-text email to be delivered to one or more
// recipients. Attachments and HTML content can be layered in the future as
// needed.
type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
}

// Sender delivers email messages via a specific transport (SMTP, third-party API,
// etc).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
