package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

// Sender sends emails via SMTP with STARTTLS.
type Sender struct {
	host string
	port string
	user string
	pass string
}

// NewSender creates an SMTP Sender.
func NewSender(host, port, user, pass string) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass}
}

// SendVerifyCode sends a verification code email to the given address.
func (s *Sender) SendVerifyCode(to, code string) error {
	addr := net.JoinHostPort(s.host, s.port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	// STARTTLS
	tlsCfg := &tls.Config{ServerName: s.host}
	if err := c.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := c.Mail(s.user); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: Rakutao Verification Code\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"<html><body>"+
		"<h2>Your Verification Code</h2>"+
		"<p style=\"font-size:32px;font-weight:bold;letter-spacing:8px;color:#333;\">%s</p>"+
		"<p>This code is valid for 10 minutes. If you did not request this, please ignore this email.</p>"+
		"<br><p style=\"color:#999;font-size:12px;\">Rakutao Collection</p>"+
		"</body></html>",
		s.user, to, code)

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return c.Quit()
}
