package mcpserver

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
)

func safeToolError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("timeout")
	}

	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		return errors.New("timeout")
	}

	var apiError *api.APIError
	if errors.As(err, &apiError) {
		switch apiError.Code {
		case api.ErrorCodeUnauthorized:
			return errors.New("authentication failure")
		case api.ErrorCodeInsufficientScope:
			return errors.New("authorization or scope failure")
		case api.ErrorCodeInvalidRequest,
			api.ErrorCodePayloadTooLarge,
			api.ErrorCodeUnsupportedMediaType:
			return errors.New("validation failure")
		case api.ErrorCodeNotFound:
			return errors.New("not found")
		case api.ErrorCodeRateLimited:
			return errors.New("rate limited")
		}

		switch apiError.StatusCode {
		case http.StatusBadRequest,
			http.StatusRequestEntityTooLarge,
			http.StatusUnsupportedMediaType,
			http.StatusUnprocessableEntity:
			return errors.New("validation failure")
		case http.StatusUnauthorized:
			return errors.New("authentication failure")
		case http.StatusForbidden:
			return errors.New("authorization or scope failure")
		case http.StatusNotFound:
			return errors.New("not found")
		case http.StatusConflict:
			return errors.New("conflict")
		case http.StatusTooManyRequests:
			return errors.New("rate limited")
		}

		if apiError.StatusCode >= 500 {
			return errors.New("PostMyForm API server failure")
		}

		return errors.New("PostMyForm API request failed")
	}

	var transportError *api.TransportError
	if errors.As(err, &transportError) {
		return errors.New("network failure")
	}

	switch {
	case errors.Is(err, api.ErrMalformedResponse),
		errors.Is(err, api.ErrUnexpectedContentType):
		return errors.New("malformed or unexpected PostMyForm API response")
	case errors.Is(err, api.ErrResponseTooLarge):
		return errors.New("PostMyForm API response exceeded the size limit")
	default:
		return errors.New("PostMyForm operation failed")
	}
}
