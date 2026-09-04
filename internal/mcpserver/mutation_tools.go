package mcpserver

import (
	"context"
	"errors"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type createFormInput struct {
	Name               string    `json:"name" jsonschema:"Form name"`
	DestinationEmail   string    `json:"destination_email" jsonschema:"Email destination for submission notifications"`
	AllowedOrigins     *[]string `json:"allowed_origins,omitempty" jsonschema:"Optional browser submission origins"`
	SuccessRedirectURL *string   `json:"success_redirect_url,omitempty" jsonschema:"Optional absolute HTTP or HTTPS success redirect URL"`
}

type updateFormInput struct {
	FormID               string    `json:"form_id" jsonschema:"PostMyForm form UUID"`
	Name                 *string   `json:"name,omitempty" jsonschema:"New form name"`
	DestinationEmail     *string   `json:"destination_email,omitempty" jsonschema:"New email destination"`
	AllowedOrigins       *[]string `json:"allowed_origins,omitempty" jsonschema:"Complete origin list; use an empty array to clear origins"`
	SuccessRedirectURL   *string   `json:"success_redirect_url,omitempty" jsonschema:"New absolute HTTP or HTTPS success redirect URL"`
	ClearSuccessRedirect bool      `json:"clear_success_redirect,omitempty" jsonschema:"Set true to explicitly clear the configured success redirect"`
	SpamHoneypotField    *string   `json:"spam_honeypot_field,omitempty" jsonschema:"New honeypot field name"`
	Status               *string   `json:"status,omitempty" jsonschema:"New editable status: active or paused"`
}

type replaceFormFieldInput struct {
	Name      string   `json:"name" jsonschema:"HTML field name"`
	Label     string   `json:"label" jsonschema:"Human-readable field label"`
	FieldType string   `json:"field_type" jsonschema:"Field type: text, email, textarea, select, or checkbox"`
	Required  bool     `json:"required" jsonschema:"Whether the generated field is required"`
	Options   []string `json:"options" jsonschema:"Complete option list; required for select fields and empty for non-select fields"`
}

type replaceFormFieldsInput struct {
	FormID string                  `json:"form_id" jsonschema:"PostMyForm form UUID"`
	Fields []replaceFormFieldInput `json:"fields" jsonschema:"Complete ordered replacement field collection"`
}

func registerMutationTools(server *mcp.Server, client APIClient) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "create_form",
			Title:       "Create a PostMyForm form",
			Description: "Create a new PostMyForm form. This operation changes server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPointer(false),
				IdempotentHint:  false,
				OpenWorldHint:   boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input createFormInput) (*mcp.CallToolResult, createFormOutput, error) {
			request, err := buildCreateFormRequest(input)
			if err != nil {
				return nil, createFormOutput{}, err
			}

			form, err := client.CreateForm(ctx, request)
			if err != nil {
				return nil, createFormOutput{}, safeToolError(err)
			}

			return nil, createFormOutput{
				Form: toFormOutput(form),
			}, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "update_form",
			Title:       "Update a PostMyForm form",
			Description: "Update only the requested properties of an existing PostMyForm form. This operation changes server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPointer(true),
				IdempotentHint:  true,
				OpenWorldHint:   boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input updateFormInput) (*mcp.CallToolResult, updateFormOutput, error) {
			formID, request, err := buildUpdateFormRequest(input)
			if err != nil {
				return nil, updateFormOutput{}, err
			}

			receipt, err := client.UpdateForm(ctx, formID, request)
			if err != nil {
				return nil, updateFormOutput{}, safeToolError(err)
			}

			return nil, toUpdateFormOutput(receipt), nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "replace_form_fields",
			Title:       "Replace PostMyForm form fields",
			Description: "Replace the complete ordered field collection for a PostMyForm form. This operation changes server state.",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    false,
				DestructiveHint: boolPointer(true),
				IdempotentHint:  true,
				OpenWorldHint:   boolPointer(false),
			},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, input replaceFormFieldsInput) (*mcp.CallToolResult, replaceFormFieldsOutput, error) {
			formID, request, err := buildReplaceFormFieldsRequest(input)
			if err != nil {
				return nil, replaceFormFieldsOutput{}, err
			}

			fields, err := client.ReplaceFormFields(ctx, formID, request)
			if err != nil {
				return nil, replaceFormFieldsOutput{}, safeToolError(err)
			}

			return nil, replaceFormFieldsOutput{
				Fields: toFormFieldOutputs(fields),
			}, nil
		},
	)
}

func buildCreateFormRequest(input createFormInput) (api.CreateFormRequest, error) {
	if err := validateRequiredText("name", input.Name, 120); err != nil {
		return api.CreateFormRequest{}, err
	}
	if err := validateRequiredText("destination_email", input.DestinationEmail, 320); err != nil {
		return api.CreateFormRequest{}, err
	}
	if input.AllowedOrigins != nil {
		if err := validateOrigins(*input.AllowedOrigins); err != nil {
			return api.CreateFormRequest{}, err
		}
	}
	if input.SuccessRedirectURL != nil {
		if err := validateURL("success_redirect_url", *input.SuccessRedirectURL); err != nil {
			return api.CreateFormRequest{}, err
		}
	}

	request := api.CreateFormRequest{
		Name:             input.Name,
		DestinationEmail: openapi_types.Email(input.DestinationEmail),
		AllowedOrigins:   input.AllowedOrigins,
	}

	if input.SuccessRedirectURL != nil {
		request.SuccessRedirectUrl = nullable.NewNullableWithValue(*input.SuccessRedirectURL)
	}

	return request, nil
}

func buildUpdateFormRequest(input updateFormInput) (api.FormId, api.PatchFormRequest, error) {
	formID, err := parseFormID(input.FormID)
	if err != nil {
		return api.FormId{}, api.PatchFormRequest{}, err
	}

	if input.SuccessRedirectURL != nil && input.ClearSuccessRedirect {
		return api.FormId{}, api.PatchFormRequest{}, errors.New(
			"invalid tool arguments: success_redirect_url and clear_success_redirect cannot be used together",
		)
	}

	propertyCount := 0
	request := api.PatchFormRequest{}

	if input.Name != nil {
		propertyCount++
		if err := validateRequiredText("name", *input.Name, 120); err != nil {
			return api.FormId{}, api.PatchFormRequest{}, err
		}
		request.Name = input.Name
	}

	if input.DestinationEmail != nil {
		propertyCount++
		if err := validateRequiredText("destination_email", *input.DestinationEmail, 320); err != nil {
			return api.FormId{}, api.PatchFormRequest{}, err
		}
		value := openapi_types.Email(*input.DestinationEmail)
		request.DestinationEmail = &value
	}

	if input.AllowedOrigins != nil {
		propertyCount++
		if err := validateOrigins(*input.AllowedOrigins); err != nil {
			return api.FormId{}, api.PatchFormRequest{}, err
		}
		request.AllowedOrigins = input.AllowedOrigins
	}

	if input.SuccessRedirectURL != nil {
		propertyCount++
		if err := validateURL("success_redirect_url", *input.SuccessRedirectURL); err != nil {
			return api.FormId{}, api.PatchFormRequest{}, err
		}
		request.SuccessRedirectUrl = nullable.NewNullableWithValue(*input.SuccessRedirectURL)
	}

	if input.ClearSuccessRedirect {
		propertyCount++
		request.SuccessRedirectUrl = nullable.NewNullNullable[string]()
	}

	if input.SpamHoneypotField != nil {
		propertyCount++
		if !fieldNamePattern.MatchString(*input.SpamHoneypotField) {
			return api.FormId{}, api.PatchFormRequest{}, errors.New(
				"invalid tool arguments: spam_honeypot_field must match ^[A-Za-z0-9_-]{1,80}$",
			)
		}
		request.SpamHoneypotField = input.SpamHoneypotField
	}

	if input.Status != nil {
		propertyCount++
		status := api.EditableFormStatus(*input.Status)
		if !status.Valid() {
			return api.FormId{}, api.PatchFormRequest{}, errors.New(
				"invalid tool arguments: status must be active or paused",
			)
		}
		request.Status = &status
	}

	if propertyCount == 0 {
		return api.FormId{}, api.PatchFormRequest{}, errors.New(
			"invalid tool arguments: update_form requires at least one property to change",
		)
	}

	return formID, request, nil
}

func buildReplaceFormFieldsRequest(
	input replaceFormFieldsInput,
) (api.FormId, api.ReplaceFormFieldsRequest, error) {
	formID, err := parseFormID(input.FormID)
	if err != nil {
		return api.FormId{}, api.ReplaceFormFieldsRequest{}, err
	}

	if len(input.Fields) < 1 || len(input.Fields) > 50 {
		return api.FormId{}, api.ReplaceFormFieldsRequest{}, errors.New(
			"invalid tool arguments: fields must contain between 1 and 50 items",
		)
	}

	fields := make([]api.ReplaceFormField, 0, len(input.Fields))

	for _, field := range input.Fields {
		if err := validateFieldInput(field); err != nil {
			return api.FormId{}, api.ReplaceFormFieldsRequest{}, err
		}

		fields = append(fields, api.ReplaceFormField{
			Name:      field.Name,
			Label:     field.Label,
			FieldType: api.FormFieldType(field.FieldType),
			Required:  field.Required,
			Options:   nullable.NewNullableWithValue(field.Options),
		})
	}

	return formID, api.ReplaceFormFieldsRequest{
		Fields: fields,
	}, nil
}
