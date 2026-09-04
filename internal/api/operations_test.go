package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"
)

const (
	testFormID   = "11111111-1111-4111-8111-111111111111"
	testFormJSON = `{
		"id":"11111111-1111-4111-8111-111111111111",
		"name":"Fixture form",
		"slug":"fixture-form",
		"status":"active",
		"endpointId":"fixture-endpoint",
		"submissionUrl":"https://postmyform.com/f/fixture-endpoint",
		"destinationEmail":"fixture@example.com",
		"allowedOrigins":["https://example.com"],
		"successRedirectUrl":null,
		"spamHoneypotField":"spamHoneypotField",
		"minSubmitSeconds":3,
		"createdAt":"2026-09-04T12:00:00Z",
		"updatedAt":"2026-09-04T12:00:00Z"
	}`
)

func mustTestFormID(t *testing.T) FormId {
	t.Helper()

	id, err := uuid.Parse(testFormID)
	if err != nil {
		t.Fatalf("uuid.Parse() error = %v", err)
	}

	return id
}

func TestPublicOperationsUseExpectedMethodsAndPaths(t *testing.T) {
	t.Parallel()

	formID := mustTestFormID(t)

	tests := []struct {
		name     string
		method   string
		path     string
		status   int
		response string
		call     func(context.Context, *Client) error
	}{
		{
			name:     "list forms",
			method:   http.MethodGet,
			path:     "/forms",
			status:   http.StatusOK,
			response: `{"data":[]}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.ListForms(ctx)
				return err
			},
		},
		{
			name:     "get form",
			method:   http.MethodGet,
			path:     "/forms/" + testFormID,
			status:   http.StatusOK,
			response: `{"data":` + testFormJSON + `}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetForm(ctx, formID)
				return err
			},
		},
		{
			name:     "create form",
			method:   http.MethodPost,
			path:     "/forms",
			status:   http.StatusCreated,
			response: `{"data":` + testFormJSON + `}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.CreateForm(ctx, CreateFormRequest{
					Name:             "Fixture form",
					DestinationEmail: "fixture@example.com",
				})
				return err
			},
		},
		{
			name:   "update form",
			method: http.MethodPatch,
			path:   "/forms/" + testFormID,
			status: http.StatusOK,
			response: `{
				"data":{
					"id":"11111111-1111-4111-8111-111111111111",
					"status":"active",
					"updatedAt":"2026-09-04T12:01:00Z"
				}
			}`,
			call: func(ctx context.Context, client *Client) error {
				name := "Updated fixture form"
				_, err := client.UpdateForm(ctx, formID, PatchFormRequest{Name: &name})
				return err
			},
		},
		{
			name:     "get form fields",
			method:   http.MethodGet,
			path:     "/forms/" + testFormID + "/fields",
			status:   http.StatusOK,
			response: `{"data":[]}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.ListFormFields(ctx, formID)
				return err
			},
		},
		{
			name:     "replace form fields",
			method:   http.MethodPut,
			path:     "/forms/" + testFormID + "/fields",
			status:   http.StatusOK,
			response: `{"data":[]}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.ReplaceFormFields(
					ctx,
					formID,
					ReplaceFormFieldsRequest{Fields: []ReplaceFormField{}},
				)
				return err
			},
		},
		{
			name:     "get form snippet",
			method:   http.MethodGet,
			path:     "/forms/" + testFormID + "/snippet",
			status:   http.StatusOK,
			response: `{"data":{"html":"<form></form>"}}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetFormSnippet(ctx, formID)
				return err
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != test.method {
					t.Fatalf("method = %q, want %q", r.Method, test.method)
				}
				if r.URL.Path != test.path {
					t.Fatalf("path = %q, want %q", r.URL.Path, test.path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", 0)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if err := test.call(context.Background(), client); err != nil {
				t.Fatalf("operation error = %v", err)
			}
		})
	}
}

func TestUpdateFormPreservesRedirectOmitAndClearSemantics(t *testing.T) {
	t.Parallel()

	formID := mustTestFormID(t)

	tests := []struct {
		name    string
		request PatchFormRequest
		check   func(*testing.T, map[string]json.RawMessage)
	}{
		{
			name:    "omit redirect",
			request: PatchFormRequest{},
			check: func(t *testing.T, body map[string]json.RawMessage) {
				t.Helper()

				if _, ok := body["successRedirectUrl"]; ok {
					t.Fatal("successRedirectUrl present, want omitted")
				}
			},
		},
		{
			name: "clear redirect",
			request: PatchFormRequest{
				SuccessRedirectUrl: nullable.NewNullNullable[string](),
			},
			check: func(t *testing.T, body map[string]json.RawMessage) {
				t.Helper()

				value, ok := body["successRedirectUrl"]
				if !ok {
					t.Fatal("successRedirectUrl omitted, want explicit null")
				}
				if strings.TrimSpace(string(value)) != "null" {
					t.Fatalf("successRedirectUrl = %s, want null", value)
				}
			},
		},
		{
			name: "set redirect",
			request: PatchFormRequest{
				SuccessRedirectUrl: nullable.NewNullableWithValue(
					"https://example.com/thanks",
				),
			},
			check: func(t *testing.T, body map[string]json.RawMessage) {
				t.Helper()

				var value string
				if err := json.Unmarshal(body["successRedirectUrl"], &value); err != nil {
					t.Fatalf("decode successRedirectUrl: %v", err)
				}
				if value != "https://example.com/thanks" {
					t.Fatalf("successRedirectUrl = %q", value)
				}
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}

				test.check(t, body)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"data":{
						"id":"11111111-1111-4111-8111-111111111111",
						"status":"active",
						"updatedAt":"2026-09-04T12:01:00Z"
					}
				}`))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", 0)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if _, err := client.UpdateForm(
				context.Background(),
				formID,
				test.request,
			); err != nil {
				t.Fatalf("UpdateForm() error = %v", err)
			}
		})
	}
}

func TestMutationOperationsDoNotRetry(t *testing.T) {
	t.Parallel()

	formID := mustTestFormID(t)

	tests := []struct {
		name   string
		method string
		path   string
		call   func(context.Context, *Client) error
	}{
		{
			name:   "create form",
			method: http.MethodPost,
			path:   "/forms",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.CreateForm(ctx, CreateFormRequest{
					Name:             "No retry",
					DestinationEmail: "retry@example.com",
				})
				return err
			},
		},
		{
			name:   "update form",
			method: http.MethodPatch,
			path:   "/forms/" + testFormID,
			call: func(ctx context.Context, client *Client) error {
				name := "No retry update"
				_, err := client.UpdateForm(
					ctx,
					formID,
					PatchFormRequest{Name: &name},
				)
				return err
			},
		},
		{
			name:   "replace form fields",
			method: http.MethodPut,
			path:   "/forms/" + testFormID + "/fields",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.ReplaceFormFields(
					ctx,
					formID,
					ReplaceFormFieldsRequest{
						Fields: []ReplaceFormField{
							{
								Name:      "email",
								Label:     "Email",
								FieldType: Email,
								Required:  true,
								Options:   nullable.NewNullableWithValue([]string{}),
							},
						},
					},
				)
				return err
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var requestCount atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount.Add(1)

				if r.Method != test.method {
					t.Errorf("method = %q, want %q", r.Method, test.method)
				}
				if r.URL.Path != test.path {
					t.Errorf("path = %q, want %q", r.URL.Path, test.path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{
					"error":{
						"code":"organization_unavailable",
						"message":"service temporarily unavailable"
					}
				}`))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", 0)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			err = test.call(context.Background(), client)
			if err == nil {
				t.Fatal("mutation error = nil, want server failure")
			}

			if got := requestCount.Load(); got != 1 {
				t.Fatalf("request count = %d, want exactly 1", got)
			}
		})
	}
}
