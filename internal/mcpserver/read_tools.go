package mcpserver

import (
	"context"
	"fmt"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listFormsInput struct{}

type formIDInput struct {
	FormID string `json:"form_id" jsonschema:"PostMyForm form UUID"`
}

func registerReadTools(server *mcp.Server, client APIClient) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_forms",
			Title:       "List PostMyForm forms",
			Description: "List forms visible to the configured PostMyForm API credential. This operation does not change server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ listFormsInput) (*mcp.CallToolResult, listFormsOutput, error) {
			forms, err := client.ListForms(ctx)
			if err != nil {
				return nil, listFormsOutput{}, safeToolError(err)
			}

			return nil, listFormsOutput{
				Forms: toFormOutputs(forms),
			}, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_form",
			Title:       "Get a PostMyForm form",
			Description: "Retrieve one form by its public PostMyForm form ID. This operation does not change server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input formIDInput) (*mcp.CallToolResult, getFormOutput, error) {
			formID, err := parseFormID(input.FormID)
			if err != nil {
				return nil, getFormOutput{}, err
			}

			form, err := client.GetForm(ctx, formID)
			if err != nil {
				return nil, getFormOutput{}, safeToolError(err)
			}

			return nil, getFormOutput{
				Form: toFormOutput(form),
			}, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_form_fields",
			Title:       "Get PostMyForm form fields",
			Description: "Retrieve the complete ordered field configuration for one form. This operation does not change server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input formIDInput) (*mcp.CallToolResult, getFormFieldsOutput, error) {
			formID, err := parseFormID(input.FormID)
			if err != nil {
				return nil, getFormFieldsOutput{}, err
			}

			fields, err := client.ListFormFields(ctx, formID)
			if err != nil {
				return nil, getFormFieldsOutput{}, safeToolError(err)
			}

			return nil, getFormFieldsOutput{
				Fields: toFormFieldOutputs(fields),
			}, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_form_snippet",
			Title:       "Get a PostMyForm form snippet",
			Description: "Retrieve the generated public form snippet as data. The MCP server does not execute returned HTML or JavaScript. This operation does not change server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input formIDInput) (*mcp.CallToolResult, getFormSnippetOutput, error) {
			formID, err := parseFormID(input.FormID)
			if err != nil {
				return nil, getFormSnippetOutput{}, err
			}

			snippet, err := client.GetFormSnippet(ctx, formID)
			if err != nil {
				return nil, getFormSnippetOutput{}, safeToolError(err)
			}

			return nil, getFormSnippetOutput{
				HTML: snippet.Html,
			}, nil
		},
	)
}

func parseFormID(value string) (api.FormId, error) {
	formID, err := uuid.Parse(value)
	if err != nil {
		return api.FormId{}, fmt.Errorf("invalid tool arguments: form_id must be a UUID")
	}

	return formID, nil
}
