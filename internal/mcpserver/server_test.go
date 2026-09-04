package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testFormID = "11111111-1111-4111-8111-111111111111"

type testAPIClient struct {
	listFormsCalls  int
	listFormsErr    error
	createRequests  []api.CreateFormRequest
	updateIDs       []api.FormId
	updateRequests  []api.PatchFormRequest
	replaceIDs      []api.FormId
	replaceRequests []api.ReplaceFormFieldsRequest
}

func (c *testAPIClient) ListForms(context.Context) ([]api.Form, error) {
	c.listFormsCalls++
	if c.listFormsErr != nil {
		return nil, c.listFormsErr
	}
	return []api.Form{}, nil
}

func (*testAPIClient) GetForm(context.Context, api.FormId) (api.Form, error) {
	return api.Form{}, nil
}

func (c *testAPIClient) CreateForm(
	_ context.Context,
	request api.CreateFormRequest,
) (api.Form, error) {
	c.createRequests = append(c.createRequests, request)
	return api.Form{}, nil
}

func (c *testAPIClient) UpdateForm(
	_ context.Context,
	formID api.FormId,
	request api.PatchFormRequest,
) (api.FormMutationReceipt, error) {
	c.updateIDs = append(c.updateIDs, formID)
	c.updateRequests = append(c.updateRequests, request)
	return api.FormMutationReceipt{}, nil
}

func (*testAPIClient) ListFormFields(context.Context, api.FormId) ([]api.FormField, error) {
	return nil, nil
}

func (c *testAPIClient) ReplaceFormFields(
	_ context.Context,
	formID api.FormId,
	request api.ReplaceFormFieldsRequest,
) ([]api.FormField, error) {
	c.replaceIDs = append(c.replaceIDs, formID)
	c.replaceRequests = append(c.replaceRequests, request)
	return nil, nil
}

func (*testAPIClient) GetFormSnippet(context.Context, api.FormId) (api.FormSnippet, error) {
	return api.FormSnippet{}, nil
}

func TestReadToolsThroughMCPProtocol(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiClient := &testAPIClient{}
	server := New("test", apiClient)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	toolsResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	wantTools := map[string]struct {
		found    bool
		readOnly bool
	}{
		"list_forms":          {readOnly: true},
		"get_form":            {readOnly: true},
		"get_form_fields":     {readOnly: true},
		"get_form_snippet":    {readOnly: true},
		"create_form":         {readOnly: false},
		"update_form":         {readOnly: false},
		"replace_form_fields": {readOnly: false},
	}

	for _, tool := range toolsResult.Tools {
		expected, ok := wantTools[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}

		expected.found = true
		wantTools[tool.Name] = expected

		if tool.InputSchema == nil {
			t.Fatalf("tool %q has no input schema", tool.Name)
		}
		if tool.OutputSchema == nil {
			t.Fatalf("tool %q has no output schema", tool.Name)
		}
		if tool.Annotations == nil {
			t.Fatalf("tool %q has no annotations", tool.Name)
		}
		if tool.Annotations.ReadOnlyHint != expected.readOnly {
			t.Fatalf(
				"tool %q ReadOnlyHint = %v, want %v",
				tool.Name,
				tool.Annotations.ReadOnlyHint,
				expected.readOnly,
			)
		}
	}

	for name, expected := range wantTools {
		if !expected.found {
			t.Fatalf("tool %q not advertised", name)
		}
	}

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_forms",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_forms) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(list_forms) returned tool error: %v", result.Content)
	}
	if apiClient.listFormsCalls != 1 {
		t.Fatalf("ListForms calls = %d, want 1", apiClient.listFormsCalls)
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_form",
		Arguments: map[string]any{
			"form_id": "not-a-uuid",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_form) protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool(get_form) IsError = false, want true")
	}
}

func TestMutationToolsThroughMCPProtocol(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiClient := &testAPIClient{}
	server := New("test", apiClient)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_form",
		Arguments: map[string]any{
			"name":                 "Contact",
			"destination_email":    "owner@example.com",
			"allowed_origins":      []string{"https://example.com"},
			"success_redirect_url": "https://example.com/thanks",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_form) protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create_form) returned tool error: %v", result.Content)
	}
	if len(apiClient.createRequests) != 1 {
		t.Fatalf("CreateForm calls = %d, want 1", len(apiClient.createRequests))
	}

	createRequest := apiClient.createRequests[0]
	if createRequest.Name != "Contact" {
		t.Fatalf("CreateForm name = %q, want Contact", createRequest.Name)
	}
	if string(createRequest.DestinationEmail) != "owner@example.com" {
		t.Fatalf(
			"CreateForm destination email = %q, want owner@example.com",
			createRequest.DestinationEmail,
		)
	}
	if createRequest.AllowedOrigins == nil ||
		len(*createRequest.AllowedOrigins) != 1 ||
		(*createRequest.AllowedOrigins)[0] != "https://example.com" {
		t.Fatalf("CreateForm allowed origins = %#v", createRequest.AllowedOrigins)
	}
	if !createRequest.SuccessRedirectUrl.IsSpecified() ||
		createRequest.SuccessRedirectUrl.IsNull() ||
		createRequest.SuccessRedirectUrl.MustGet() != "https://example.com/thanks" {
		t.Fatal("CreateForm success redirect was not preserved")
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_form",
		Arguments: map[string]any{
			"form_id": testFormID,
			"name":    "Updated Contact",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(update_form omit redirect) protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(update_form omit redirect) returned tool error: %v", result.Content)
	}
	if len(apiClient.updateRequests) != 1 {
		t.Fatalf("UpdateForm calls = %d, want 1", len(apiClient.updateRequests))
	}
	if apiClient.updateRequests[0].SuccessRedirectUrl.IsSpecified() {
		t.Fatal("UpdateForm omitted redirect became specified")
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_form",
		Arguments: map[string]any{
			"form_id":                testFormID,
			"clear_success_redirect": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(update_form clear redirect) protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(update_form clear redirect) returned tool error: %v", result.Content)
	}
	if len(apiClient.updateRequests) != 2 {
		t.Fatalf("UpdateForm calls = %d, want 2", len(apiClient.updateRequests))
	}

	clearRequest := apiClient.updateRequests[1]
	if !clearRequest.SuccessRedirectUrl.IsSpecified() ||
		!clearRequest.SuccessRedirectUrl.IsNull() {
		t.Fatal("UpdateForm explicit redirect clear was not preserved")
	}
	if apiClient.updateIDs[1].String() != testFormID {
		t.Fatalf("UpdateForm form ID = %s, want %s", apiClient.updateIDs[1], testFormID)
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_form",
		Arguments: map[string]any{
			"form_id":                testFormID,
			"success_redirect_url":   "https://example.com/thanks",
			"clear_success_redirect": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(update_form conflicting redirect) protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool(update_form conflicting redirect) IsError = false, want true")
	}
	if len(apiClient.updateRequests) != 2 {
		t.Fatalf(
			"UpdateForm calls after rejected input = %d, want 2",
			len(apiClient.updateRequests),
		)
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "replace_form_fields",
		Arguments: map[string]any{
			"form_id": testFormID,
			"fields": []map[string]any{
				{
					"name":       "email",
					"label":      "Email",
					"field_type": "email",
					"required":   true,
					"options":    []string{},
				},
				{
					"name":       "topic",
					"label":      "Topic",
					"field_type": "select",
					"required":   false,
					"options":    []string{"Sales", "Support"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(replace_form_fields) protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(replace_form_fields) returned tool error: %v", result.Content)
	}
	if len(apiClient.replaceRequests) != 1 {
		t.Fatalf("ReplaceFormFields calls = %d, want 1", len(apiClient.replaceRequests))
	}
	if apiClient.replaceIDs[0].String() != testFormID {
		t.Fatalf(
			"ReplaceFormFields form ID = %s, want %s",
			apiClient.replaceIDs[0],
			testFormID,
		)
	}

	fields := apiClient.replaceRequests[0].Fields
	if len(fields) != 2 {
		t.Fatalf("ReplaceFormFields field count = %d, want 2", len(fields))
	}
	if fields[1].FieldType != api.Select {
		t.Fatalf("second field type = %q, want select", fields[1].FieldType)
	}
	if !fields[1].Options.IsSpecified() || fields[1].Options.IsNull() {
		t.Fatal("select field options were not preserved")
	}

	options := fields[1].Options.MustGet()
	if len(options) != 2 || options[0] != "Sales" || options[1] != "Support" {
		t.Fatalf("select field options = %#v", options)
	}

	result, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "replace_form_fields",
		Arguments: map[string]any{
			"form_id": testFormID,
			"fields":  []map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(replace_form_fields empty) protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool(replace_form_fields empty) IsError = false, want true")
	}
	if len(apiClient.replaceRequests) != 1 {
		t.Fatalf(
			"ReplaceFormFields calls after rejected input = %d, want 1",
			len(apiClient.replaceRequests),
		)
	}
}

func TestMCPProtocolSanitizesUpstreamErrors(t *testing.T) {
	t.Parallel()

	const sentinel = "pmf_secret_protocol_777777"

	ctx := context.Background()
	apiClient := &testAPIClient{
		listFormsErr: &api.APIError{
			StatusCode: 401,
			Code:       api.ErrorCodeUnauthorized,
			Message:    sentinel,
			RetryAfter: sentinel,
		},
	}
	server := New("test", apiClient)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_forms",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_forms) protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool(list_forms) IsError = false, want true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("CallTool(list_forms) content count = %d, want 1", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf(
			"CallTool(list_forms) content type = %T, want *mcp.TextContent",
			result.Content[0],
		)
	}

	if textContent.Text != "authentication failure" {
		t.Fatalf(
			"CallTool(list_forms) error = %q, want authentication failure",
			textContent.Text,
		)
	}
	if strings.Contains(textContent.Text, sentinel) {
		t.Fatal("CallTool(list_forms) leaked credential sentinel")
	}
}

func TestToolInputSchemasExposeOnlyApprovedArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := New("test", &testAPIClient{})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-schema-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	toolsResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	wantProperties := map[string][]string{
		"list_forms":          {},
		"get_form":            {"form_id"},
		"get_form_fields":     {"form_id"},
		"get_form_snippet":    {"form_id"},
		"create_form":         {"allowed_origins", "destination_email", "name", "success_redirect_url"},
		"update_form":         {"allowed_origins", "clear_success_redirect", "destination_email", "form_id", "name", "spam_honeypot_field", "status", "success_redirect_url"},
		"replace_form_fields": {"fields", "form_id"},
	}

	for _, tool := range toolsResult.Tools {
		want, ok := wantProperties[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}

		schema, ok := tool.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("tool %q input schema type = %T, want map[string]any", tool.Name, tool.InputSchema)
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			if len(want) == 0 && schema["properties"] == nil {
				continue
			}
			t.Fatalf("tool %q properties type = %T, want map[string]any", tool.Name, schema["properties"])
		}

		if len(properties) != len(want) {
			t.Fatalf(
				"tool %q property count = %d, want %d: %#v",
				tool.Name,
				len(properties),
				len(want),
				properties,
			)
		}

		for _, name := range want {
			if _, ok := properties[name]; !ok {
				t.Fatalf("tool %q missing approved property %q", tool.Name, name)
			}
		}

		for name := range properties {
			if !containsString(want, name) {
				t.Fatalf("tool %q exposes unapproved property %q", tool.Name, name)
			}
		}
	}
}

func TestToolSchemasDoNotExposeGenericDangerousInputs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := New("test", &testAPIClient{})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-security-schema-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	toolsResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	forbidden := map[string]struct{}{
		"token":         {},
		"api_token":     {},
		"authorization": {},
		"headers":       {},
		"header":        {},
		"method":        {},
		"body":          {},
		"url":           {},
		"base_url":      {},
		"command":       {},
		"shell":         {},
		"file":          {},
		"filename":      {},
		"path":          {},
	}

	for _, tool := range toolsResult.Tools {
		assertNoForbiddenSchemaProperties(t, tool.Name, tool.InputSchema, forbidden)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertNoForbiddenSchemaProperties(
	t *testing.T,
	toolName string,
	value any,
	forbidden map[string]struct{},
) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		if properties, ok := typed["properties"].(map[string]any); ok {
			for name, propertySchema := range properties {
				if _, blocked := forbidden[name]; blocked {
					t.Fatalf("tool %q exposes forbidden property %q", toolName, name)
				}
				assertNoForbiddenSchemaProperties(t, toolName, propertySchema, forbidden)
			}
		}

		if items, ok := typed["items"]; ok {
			assertNoForbiddenSchemaProperties(t, toolName, items, forbidden)
		}

	case []any:
		for _, item := range typed {
			assertNoForbiddenSchemaProperties(t, toolName, item, forbidden)
		}
	}
}

type successfulReadAPIClient struct {
	*testAPIClient
	getFormIDs        []api.FormId
	getFormFieldIDs   []api.FormId
	getFormSnippetIDs []api.FormId
}

func (c *successfulReadAPIClient) GetForm(
	_ context.Context,
	formID api.FormId,
) (api.Form, error) {
	c.getFormIDs = append(c.getFormIDs, formID)
	return api.Form{}, nil
}

func (c *successfulReadAPIClient) ListFormFields(
	_ context.Context,
	formID api.FormId,
) ([]api.FormField, error) {
	c.getFormFieldIDs = append(c.getFormFieldIDs, formID)
	return []api.FormField{}, nil
}

func (c *successfulReadAPIClient) GetFormSnippet(
	_ context.Context,
	formID api.FormId,
) (api.FormSnippet, error) {
	c.getFormSnippetIDs = append(c.getFormSnippetIDs, formID)
	return api.FormSnippet{
		Html: "<form></form>",
	}, nil
}

func TestRemainingReadToolsThroughMCPProtocol(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiClient := &successfulReadAPIClient{
		testAPIClient: &testAPIClient{},
	}
	server := New("test", apiClient)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-read-test-client",
			Version: "test",
		},
		nil,
	)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	tests := []struct {
		name string
	}{
		{name: "get_form"},
		{name: "get_form_fields"},
		{name: "get_form_snippet"},
	}

	for _, test := range tests {
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name: test.name,
			Arguments: map[string]any{
				"form_id": testFormID,
			},
		})
		if err != nil {
			t.Fatalf("CallTool(%s) protocol error = %v", test.name, err)
		}
		if result.IsError {
			t.Fatalf("CallTool(%s) returned tool error: %v", test.name, result.Content)
		}
	}

	if len(apiClient.getFormIDs) != 1 ||
		apiClient.getFormIDs[0].String() != testFormID {
		t.Fatalf("GetForm IDs = %#v, want %s", apiClient.getFormIDs, testFormID)
	}

	if len(apiClient.getFormFieldIDs) != 1 ||
		apiClient.getFormFieldIDs[0].String() != testFormID {
		t.Fatalf(
			"ListFormFields IDs = %#v, want %s",
			apiClient.getFormFieldIDs,
			testFormID,
		)
	}

	if len(apiClient.getFormSnippetIDs) != 1 ||
		apiClient.getFormSnippetIDs[0].String() != testFormID {
		t.Fatalf(
			"GetFormSnippet IDs = %#v, want %s",
			apiClient.getFormSnippetIDs,
			testFormID,
		)
	}
}
