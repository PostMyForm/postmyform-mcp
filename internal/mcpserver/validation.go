package mcpserver

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
)

var fieldNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

func validateRequiredText(name, value string, maxLength int) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("invalid tool arguments: " + name + " must not be blank")
	}
	if utf8.RuneCountInString(value) > maxLength {
		return errors.New("invalid tool arguments: " + name + " is too long")
	}
	return nil
}

func validateURL(name, value string) error {
	if utf8.RuneCountInString(value) > 2048 {
		return errors.New("invalid tool arguments: " + name + " is too long")
	}

	parsed, err := url.Parse(value)
	if err != nil ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("invalid tool arguments: " + name + " must be an absolute HTTP or HTTPS URL")
	}

	return nil
}

func validateOrigins(origins []string) error {
	if len(origins) > 50 {
		return errors.New("invalid tool arguments: allowed_origins must contain at most 50 values")
	}

	for _, origin := range origins {
		if err := validateURL("allowed_origins value", origin); err != nil {
			return err
		}
	}

	return nil
}

func validateFieldInput(field replaceFormFieldInput) error {
	if !fieldNamePattern.MatchString(field.Name) {
		return errors.New("invalid tool arguments: field name must match ^[A-Za-z0-9_-]{1,80}$")
	}
	if err := validateRequiredText("field label", field.Label, 120); err != nil {
		return err
	}

	fieldType := api.FormFieldType(field.FieldType)
	if !fieldType.Valid() {
		return errors.New("invalid tool arguments: field_type must be text, email, textarea, select, or checkbox")
	}

	if len(field.Options) > 50 {
		return errors.New("invalid tool arguments: field options must contain at most 50 values")
	}

	if fieldType == api.Select {
		if len(field.Options) == 0 {
			return errors.New("invalid tool arguments: select fields require at least one option")
		}
	} else if len(field.Options) != 0 {
		return errors.New("invalid tool arguments: non-select fields must use an empty options array")
	}

	seen := make(map[string]struct{}, len(field.Options))
	for _, option := range field.Options {
		if err := validateRequiredText("field option", option, 120); err != nil {
			return err
		}

		normalized := strings.TrimSpace(option)
		if _, ok := seen[normalized]; ok {
			return errors.New("invalid tool arguments: field options must not contain duplicates")
		}
		seen[normalized] = struct{}{}
	}

	return nil
}
