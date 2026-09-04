package mcpserver

import (
	"time"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
)

type formOutput struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Status             string    `json:"status"`
	EndpointID         string    `json:"endpointId"`
	SubmissionURL      string    `json:"submissionUrl"`
	DestinationEmail   string    `json:"destinationEmail"`
	AllowedOrigins     []string  `json:"allowedOrigins"`
	SuccessRedirectURL *string   `json:"successRedirectUrl"`
	SpamHoneypotField  string    `json:"spamHoneypotField"`
	MinSubmitSeconds   int       `json:"minSubmitSeconds"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type formFieldOutput struct {
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	FieldType string    `json:"fieldType"`
	Required  bool      `json:"required"`
	Options   *[]string `json:"options"`
}

type listFormsOutput struct {
	Forms []formOutput `json:"forms"`
}

type getFormOutput struct {
	Form formOutput `json:"form"`
}

type getFormFieldsOutput struct {
	Fields []formFieldOutput `json:"fields"`
}

type getFormSnippetOutput struct {
	HTML string `json:"html"`
}

func toFormOutput(form api.Form) formOutput {
	var successRedirectURL *string

	if form.SuccessRedirectUrl.IsSpecified() && !form.SuccessRedirectUrl.IsNull() {
		value := form.SuccessRedirectUrl.MustGet()
		successRedirectURL = &value
	}

	return formOutput{
		ID:                 form.Id.String(),
		Name:               form.Name,
		Slug:               form.Slug,
		Status:             string(form.Status),
		EndpointID:         form.EndpointId,
		SubmissionURL:      form.SubmissionUrl,
		DestinationEmail:   form.DestinationEmail,
		AllowedOrigins:     form.AllowedOrigins,
		SuccessRedirectURL: successRedirectURL,
		SpamHoneypotField:  form.SpamHoneypotField,
		MinSubmitSeconds:   form.MinSubmitSeconds,
		CreatedAt:          form.CreatedAt,
		UpdatedAt:          form.UpdatedAt,
	}
}

func toFormOutputs(forms []api.Form) []formOutput {
	result := make([]formOutput, 0, len(forms))
	for _, form := range forms {
		result = append(result, toFormOutput(form))
	}
	return result
}

func toFormFieldOutput(field api.FormField) formFieldOutput {
	var options *[]string

	if field.Options.IsSpecified() && !field.Options.IsNull() {
		value := field.Options.MustGet()
		options = &value
	}

	return formFieldOutput{
		Name:      field.Name,
		Label:     field.Label,
		FieldType: string(field.FieldType),
		Required:  field.Required,
		Options:   options,
	}
}

func toFormFieldOutputs(fields []api.FormField) []formFieldOutput {
	result := make([]formFieldOutput, 0, len(fields))
	for _, field := range fields {
		result = append(result, toFormFieldOutput(field))
	}
	return result
}

type createFormOutput struct {
	Form formOutput `json:"form"`
}

type updateFormOutput struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type replaceFormFieldsOutput struct {
	Fields []formFieldOutput `json:"fields"`
}

func toUpdateFormOutput(receipt api.FormMutationReceipt) updateFormOutput {
	return updateFormOutput{
		ID:        receipt.Id.String(),
		Status:    string(receipt.Status),
		UpdatedAt: receipt.UpdatedAt,
	}
}
