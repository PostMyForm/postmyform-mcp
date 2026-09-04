package mcpserver

import (
	"context"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "postmyform-mcp"

type APIClient interface {
	ListForms(context.Context) ([]api.Form, error)
	GetForm(context.Context, api.FormId) (api.Form, error)
	CreateForm(context.Context, api.CreateFormRequest) (api.Form, error)
	UpdateForm(context.Context, api.FormId, api.PatchFormRequest) (api.FormMutationReceipt, error)
	ListFormFields(context.Context, api.FormId) ([]api.FormField, error)
	ReplaceFormFields(context.Context, api.FormId, api.ReplaceFormFieldsRequest) ([]api.FormField, error)
	GetFormSnippet(context.Context, api.FormId) (api.FormSnippet, error)
}

func New(version string, client APIClient) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    serverName,
			Version: version,
		},
		nil,
	)

	registerReadTools(server, client)
	registerMutationTools(server, client)

	return server
}

func boolPointer(value bool) *bool {
	return &value
}
