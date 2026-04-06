package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

type Message struct {
	From    string
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ResendSender struct {
	apiKey     string
	apiURL     string
	httpClient httpDoer
}

func NewResendSender(apiKey string) *ResendSender {
	return &ResendSender{
		apiKey: strings.TrimSpace(apiKey),
		apiURL: resendAPIURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ResendSender) Send(ctx context.Context, message Message) error {
	if s == nil || strings.TrimSpace(s.apiKey) == "" {
		return errors.New("resend api key is not configured")
	}

	payload, err := json.Marshal(map[string]any{
		"from":    message.From,
		"to":      message.To,
		"subject": message.Subject,
		"html":    message.HTML,
		"text":    message.Text,
	})
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build resend request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+s.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("resend returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
}
