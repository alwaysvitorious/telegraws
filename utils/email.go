package utils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

func SendToEmail(
	smtpHost string,
	smtpPort int,
	username, password,
	headerFrom, envelopeFrom, toAddr,
	bodyHTML string,
) error {
	subject := "Telegraws"
	headers := map[string]string{
		"From":         headerFrom,
		"To":           toAddr,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"utf-8\"",
	}

	var msgBuilder strings.Builder
	for k, v := range headers {
		msgBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msgBuilder.WriteString("\r\n") // header/body separator
	msgBuilder.WriteString(bodyHTML)

	rawMsg := []byte(msgBuilder.String())

	// Dial TLS
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: smtpHost,
	})
	if err != nil {
		return fmt.Errorf("smtp dial error: %w", err)
	}
	defer conn.Close()

	// New SMTP client
	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("creating smtp client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	auth := smtp.PlainAuth("", username, password, smtpHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth error: %w", err)
	}

	// MAIL FROM
	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("smtp MAIL FROM error: %w", err)
	}
	// RCPT TO
	if err := client.Rcpt(toAddr); err != nil {
		return fmt.Errorf("smtp RCPT TO error: %w", err)
	}

	// DATA
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA error: %w", err)
	}
	if _, err := wc.Write(rawMsg); err != nil {
		return fmt.Errorf("writing message data: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("closing data writer: %w", err)
	}

	return nil
}
