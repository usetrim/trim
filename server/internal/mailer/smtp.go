package mailer

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPConfig is optional outbound mail. All fields must be set to send.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Enabled reports whether SMTP is fully configured (no silent invent).
func (c SMTPConfig) Enabled() bool {
	return strings.TrimSpace(c.Host) != "" &&
		strings.TrimSpace(c.Port) != "" &&
		strings.TrimSpace(c.From) != ""
}

// SendPlain sends a plain-text email. Returns error if SMTP is disabled or send fails.
func (c SMTPConfig) SendPlain(to, subject, body string) error {
	if !c.Enabled() {
		return fmt.Errorf("smtp is not configured")
	}
	to = strings.TrimSpace(to)
	if to == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("invalid recipient")
	}

	addr := net.JoinHostPort(strings.TrimSpace(c.Host), strings.TrimSpace(c.Port))
	from := strings.TrimSpace(c.From)
	msg := strings.Builder{}
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	var auth smtp.Auth
	user := strings.TrimSpace(c.Username)
	pass := c.Password
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, strings.TrimSpace(c.Host))
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg.String()))
}
