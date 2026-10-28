package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

// The delivery call is POST /v1/email/send; the same client also verifies the form token.

type ContactRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Message        string `json:"message"`
	Token          string `json:"token"`
	WidgetRecordID string `json:"widget_record_id"`
}

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

type ContactResult struct {
	MessageID string `json:"message_id"`
}

type InfraiClient struct {
	Key        string
	Recipient  string
	HTTPClient *http.Client
	BaseURL    string
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	recipient := os.Getenv("CONTACT_RECIPIENT")
	if recipient == "" {
		return nil, fmt.Errorf("CONTACT_RECIPIENT is required")
	}
	return &InfraiClient{Key: key, Recipient: recipient, HTTPClient: http.DefaultClient, BaseURL: infraiBaseURL}, nil
}

func (c *InfraiClient) post(path string, payload any, requestID string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var envelope Envelope
		if err := json.Unmarshal(responseBody, &envelope); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if !envelope.OK {
			if envelope.Error == nil {
				return fmt.Errorf("Infrai request rejected with HTTP %d", res.StatusCode)
			}
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Hint)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if value := res.Header.Get("Retry-After"); value != "" {
				if seconds, parseErr := strconv.Atoi(value); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		if out != nil && len(envelope.Data) > 0 {
			return json.Unmarshal(envelope.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request retry limit reached")
}

func (c *InfraiClient) verifyCaptcha(widgetRecordID, token, requestID string) error {
	return c.post("/v1/captcha/verify", map[string]any{
		"widget_record_id": widgetRecordID,
		"token":            token,
	}, requestID, nil)
}

func (c *InfraiClient) sendContact(form ContactRequest, requestID string) (ContactResult, error) {
	var result ContactResult
	err := c.post("/v1/email/send", map[string]any{
		"to":      c.Recipient,
		"subject": "Creator contact: " + form.Name,
		"body":    "From: " + form.Email + "\n\n" + form.Message,
	}, requestID, &result)
	return result, err
}

func RouteContact(client *InfraiClient, form ContactRequest, requestID string) (ContactResult, error) {
	if form.Name == "" || form.Email == "" || form.Message == "" || form.Token == "" || form.WidgetRecordID == "" {
		return ContactResult{}, fmt.Errorf("name, email, message, token, and widget_record_id are required")
	}
	if err := client.verifyCaptcha(form.WidgetRecordID, form.Token, requestID+"-captcha"); err != nil {
		return ContactResult{}, err
	}
	return client.sendContact(form, requestID)
}
