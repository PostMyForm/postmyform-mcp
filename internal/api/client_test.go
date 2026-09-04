package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientRequiresToken(t *testing.T) {
	t.Parallel()

	_, err := NewClient("", "", 0)
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrMissingToken)
	}
}

func TestNewClientRejectsNonLoopbackHTTP(t *testing.T) {
	t.Parallel()

	_, err := NewClient("http://example.com/api/v1", "test-token", 0)
	if !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrInvalidBaseURL)
	}
}

func TestListFormsSendsBearerCredential(t *testing.T) {
	t.Parallel()

	const token = "sentinel-list-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("Authorization = %q, want bearer credential", got)
		}
		if got := r.URL.Path; got != "/forms" {
			t.Fatalf("path = %q, want /forms", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	forms, err := client.ListForms(context.Background())
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	if len(forms) != 0 {
		t.Fatalf("ListForms() returned %d forms, want 0", len(forms))
	}
}

func TestListFormsTimeoutReturnsTransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want timeout")
	}

	var transportError *TransportError
	if !errors.As(err, &transportError) {
		t.Fatalf("error = %T %v, want *TransportError", err, err)
	}
}

func TestListFormsRejectsUnexpectedContentType(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrUnexpectedContentType) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrUnexpectedContentType)
	}
}

func TestListFormsRejectsMissingRequiredData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("ListForms() error = %v, want %v", err, ErrMalformedResponse)
	}
}

func TestCreateFormDoesNotRetryAndRedactsCredential(t *testing.T) {
	t.Parallel()

	const token = "sentinel-create-token"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(
			`{"error":{"code":"invalid_request","message":"failure ` + token + `"}}`,
		))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.CreateForm(
		context.Background(),
		CreateFormRequest{
			Name:             "Retry test",
			DestinationEmail: "retry@example.com",
		},
	)
	if err == nil {
		t.Fatal("CreateForm() error = nil, want failure")
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("CreateForm() error exposed credential: %q", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("CreateForm() error = %q, want redaction marker", err)
	}
}

func TestAPIErrorRedactsAuthorizationAndRetryAfter(t *testing.T) {
	t.Parallel()

	const token = "777777"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", token)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w, `{
			"error":{
				"code":"rate_limited",
				"message":"request failed with Authorization: Bearer %s"
			}
		}`, token)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want failure")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
	}
	if strings.Contains(apiError.Error(), token) {
		t.Fatalf("API error exposed credential: %q", apiError.Error())
	}
	if apiError.RetryAfter != "[REDACTED]" {
		t.Fatalf("RetryAfter = %q, want %q", apiError.RetryAfter, "[REDACTED]")
	}
}

func TestAPIErrorRejectsInvalidRetryAfter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "invalid")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{
			"error":{
				"code":"rate_limited",
				"message":"Too many requests"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("ListForms() error = %T %v, want *APIError", err, err)
	}
	if apiError.RetryAfter != "" {
		t.Fatalf("RetryAfter = %q, want empty", apiError.RetryAfter)
	}
}

func TestListFormsRejectsUntrustedTLSCertificate(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want TLS verification failure")
	}

	var transportError *TransportError
	if !errors.As(err, &transportError) {
		t.Fatalf("ListForms() error = %T %v, want *TransportError", err, err)
	}
}
