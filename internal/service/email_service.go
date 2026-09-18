package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/http"
	"net/smtp"
	"strings"
)

type EmailSender interface {
	Send(ctx context.Context, message EmailMessage) (EmailDelivery, error)
}

type EmailMessage struct {
	From           string
	To             string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

type EmailDelivery struct {
	ID string `json:"id"`
}

type ResendAPIEmailSender struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewResendAPIEmailSender(apiKey string, baseURL string, client *http.Client) *ResendAPIEmailSender {
	if client == nil {
		client = http.DefaultClient
	}

	return &ResendAPIEmailSender{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: client,
	}
}

func (s *ResendAPIEmailSender) Send(ctx context.Context, message EmailMessage) (EmailDelivery, error) {
	if s.apiKey == "" {
		return EmailDelivery{}, errors.New("RESEND_API_KEY is required")
	}

	if err := message.Validate(); err != nil {
		return EmailDelivery{}, err
	}

	payload := resendEmailRequest{
		From:    message.From,
		To:      []string{message.To},
		Subject: message.Subject,
		HTML:    message.HTML,
		Text:    message.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return EmailDelivery{}, fmt.Errorf("marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return EmailDelivery{}, fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if message.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", message.IdempotencyKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return EmailDelivery{}, fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return EmailDelivery{}, fmt.Errorf("read resend response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return EmailDelivery{}, fmt.Errorf("resend returned %s: %s", resp.Status, string(respBody))
	}

	var delivery EmailDelivery
	if err := json.Unmarshal(respBody, &delivery); err != nil {
		return EmailDelivery{}, fmt.Errorf("decode resend response: %w", err)
	}

	return delivery, nil
}

type SMTPEmailSender struct {
	host        string
	port        string
	username    string
	password    string
	implicitTLS bool
}

func NewSMTPEmailSender(host string, port string, username string, password string, implicitTLS bool) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		implicitTLS: implicitTLS,
	}
}

func (s *SMTPEmailSender) Send(ctx context.Context, message EmailMessage) (EmailDelivery, error) {
	if err := message.Validate(); err != nil {
		return EmailDelivery{}, err
	}
	if s.host == "" || s.port == "" || s.username == "" || s.password == "" {
		return EmailDelivery{}, errors.New("smtp host, port, username, and password are required")
	}

	address := net.JoinHostPort(s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	rawMessage := buildSMTPMessage(message)
	fromAddress, err := parseEmailAddress(message.From, "email from")
	if err != nil {
		return EmailDelivery{}, err
	}
	toAddress, err := parseEmailAddress(message.To, "email to")
	if err != nil {
		return EmailDelivery{}, err
	}

	if s.implicitTLS {
		if err := s.sendWithImplicitTLS(ctx, address, auth, fromAddress, toAddress, rawMessage); err != nil {
			return EmailDelivery{}, err
		}
		return EmailDelivery{ID: message.IdempotencyKey}, nil
	}

	if err := smtp.SendMail(address, auth, fromAddress, []string{toAddress}, rawMessage); err != nil {
		return EmailDelivery{}, fmt.Errorf("send smtp email: %w", err)
	}

	return EmailDelivery{ID: message.IdempotencyKey}, nil
}

func (s *SMTPEmailSender) sendWithImplicitTLS(
	ctx context.Context,
	address string,
	auth smtp.Auth,
	fromAddress string,
	toAddress string,
	rawMessage []byte,
) error {
	var dialer net.Dialer
	conn, err := tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("connect smtp tls: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authenticate smtp: %w", err)
	}
	if err := client.Mail(fromAddress); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	if err := client.Rcpt(toAddress); err != nil {
		return fmt.Errorf("set smtp recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open smtp data writer: %w", err)
	}
	if _, err := writer.Write(rawMessage); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message: %w", err)
	}

	return client.Quit()
}

func (m EmailMessage) Validate() error {
	if strings.TrimSpace(m.From) == "" {
		return errors.New("email from is required")
	}
	if _, err := parseEmailAddress(m.From, "email from"); err != nil {
		return err
	}
	if strings.TrimSpace(m.To) == "" {
		return errors.New("email to is required")
	}
	if _, err := parseEmailAddress(m.To, "email to"); err != nil {
		return err
	}
	if strings.TrimSpace(m.Subject) == "" {
		return errors.New("email subject is required")
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("email subject contains invalid newline")
	}
	if strings.TrimSpace(m.HTML) == "" && strings.TrimSpace(m.Text) == "" {
		return errors.New("email html or text body is required")
	}
	return nil
}

func parseEmailAddress(value string, field string) (string, error) {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return "", fmt.Errorf("parse %s address: %w", field, err)
	}
	return address.Address, nil
}

func buildSMTPMessage(message EmailMessage) []byte {
	contentType := "text/plain; charset=\"UTF-8\""
	if message.HTML != "" {
		contentType = "text/html; charset=\"UTF-8\""
	}

	headers := []emailHeader{
		{Key: "From", Value: cleanHeaderValue(message.From)},
		{Key: "To", Value: cleanHeaderValue(message.To)},
		{Key: "Subject", Value: cleanHeaderValue(message.Subject)},
		{Key: "MIME-Version", Value: "1.0"},
		{Key: "Content-Type", Value: contentType},
	}
	if message.IdempotencyKey != "" {
		headers = append(headers, emailHeader{Key: "Resend-Idempotency-Key", Value: cleanHeaderValue(message.IdempotencyKey)})
	}

	var builder strings.Builder
	for _, header := range headers {
		builder.WriteString(header.Key)
		builder.WriteString(": ")
		builder.WriteString(header.Value)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	if message.HTML != "" {
		builder.WriteString(message.HTML)
	} else {
		builder.WriteString(message.Text)
	}

	return []byte(builder.String())
}

func cleanHeaderValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type emailHeader struct {
	Key   string
	Value string
}
