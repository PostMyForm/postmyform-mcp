package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
)

func TestSafeToolErrorCategoriesAndRedaction(t *testing.T) {
	t.Parallel()

	const sentinel = "pmf_secret_777777"

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "authentication",
			err: &api.APIError{
				StatusCode: http.StatusUnauthorized,
				Code:       api.ErrorCodeUnauthorized,
				Message:    sentinel,
				RetryAfter: sentinel,
			},
			want: "authentication failure",
		},
		{
			name: "authorization",
			err: &api.APIError{
				StatusCode: http.StatusForbidden,
				Code:       api.ErrorCodeInsufficientScope,
				Message:    sentinel,
			},
			want: "authorization or scope failure",
		},
		{
			name: "validation",
			err: &api.APIError{
				StatusCode: http.StatusBadRequest,
				Code:       api.ErrorCodeInvalidRequest,
				Message:    sentinel,
			},
			want: "validation failure",
		},
		{
			name: "not found",
			err: &api.APIError{
				StatusCode: http.StatusNotFound,
				Code:       api.ErrorCodeNotFound,
				Message:    sentinel,
			},
			want: "not found",
		},
		{
			name: "conflict",
			err: &api.APIError{
				StatusCode: http.StatusConflict,
				Message:    sentinel,
			},
			want: "conflict",
		},
		{
			name: "rate limited",
			err: &api.APIError{
				StatusCode: http.StatusTooManyRequests,
				Code:       api.ErrorCodeRateLimited,
				Message:    sentinel,
				RetryAfter: sentinel,
			},
			want: "rate limited",
		},
		{
			name: "server",
			err: &api.APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    sentinel,
			},
			want: "PostMyForm API server failure",
		},
		{
			name: "malformed response",
			err:  api.ErrMalformedResponse,
			want: "malformed or unexpected PostMyForm API response",
		},
		{
			name: "unexpected content type",
			err:  api.ErrUnexpectedContentType,
			want: "malformed or unexpected PostMyForm API response",
		},
		{
			name: "oversize response",
			err:  api.ErrResponseTooLarge,
			want: "PostMyForm API response exceeded the size limit",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := safeToolError(test.err).Error()
			if got != test.want {
				t.Fatalf("safeToolError() = %q, want %q", got, test.want)
			}
			if strings.Contains(got, sentinel) {
				t.Fatalf("safeToolError() leaked credential sentinel %q", sentinel)
			}
		})
	}
}

func TestSafeToolErrorMapsClientTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	client, err := api.NewClient(server.URL, "pmf_secret_timeout_777777", 5*time.Millisecond)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want timeout")
	}

	got := safeToolError(err).Error()
	if got != "timeout" {
		t.Fatalf("safeToolError() = %q, want timeout", got)
	}
	if strings.Contains(got, "pmf_secret_timeout_777777") {
		t.Fatal("timeout error leaked credential sentinel")
	}
}

func TestSafeToolErrorMapsUnknownTransportFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	client, err := api.NewClient(
		serverURL,
		"pmf_secret_network_777777",
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListForms(context.Background())
	if err == nil {
		t.Fatal("ListForms() error = nil, want network failure")
	}

	got := safeToolError(err).Error()
	if got != "network failure" {
		t.Fatalf("safeToolError() = %q, want network failure", got)
	}
	if strings.Contains(got, "pmf_secret_network_777777") {
		t.Fatal("network error leaked credential sentinel")
	}
}

func TestSafeToolErrorNeverReturnsUnknownErrorText(t *testing.T) {
	t.Parallel()

	const sentinel = "pmf_secret_unknown_777777"

	got := safeToolError(errors.New(sentinel)).Error()
	if got != "PostMyForm operation failed" {
		t.Fatalf("safeToolError() = %q, want generic failure", got)
	}
	if strings.Contains(got, sentinel) {
		t.Fatal("unknown error text leaked credential sentinel")
	}
}
