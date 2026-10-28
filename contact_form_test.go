package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouteContactVerifiesBeforeSending(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true,"data":{"message_id":"msg-1"}}`)
	}))
	defer server.Close()

	client := &InfraiClient{Key: "test-key", Recipient: "chenhua@changba.com", HTTPClient: server.Client(), BaseURL: server.URL}
	result, err := RouteContact(client, ContactRequest{Name: "Mina", Email: "mina@example.com", Message: "Hello", Token: "captcha-token", WidgetRecordID: "widget-1"}, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "msg-1" {
		t.Fatalf("message id = %q", result.MessageID)
	}
	want := []string{"/v1/captcha/verify", "/v1/email/send"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestRouteContactStopsOnCaptchaRejection(t *testing.T) {
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":false,"error":{"code":"CAPTCHA_REJECTED","hint":"try again"}}`)
	}))
	defer server.Close()

	client := &InfraiClient{Key: "test-key", Recipient: "chenhua@changba.com", HTTPClient: server.Client(), BaseURL: server.URL}
	_, err := RouteContact(client, ContactRequest{Name: "Mina", Email: "mina@example.com", Message: "Hello", Token: "bad-token", WidgetRecordID: "widget-2"}, "request-2")
	if err == nil || called != 1 {
		t.Fatalf("err = %v, calls = %d", err, called)
	}
}
