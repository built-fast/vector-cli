package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// APIError represents a structured error from the Vector API.
type APIError struct {
	Code             string              `json:"code"`
	Message          string              `json:"message"`
	HTTPStatus       int                 `json:"http_status"`
	ExitCode         int                 `json:"exit_code"`
	ValidationErrors map[string][]string `json:"validation_errors,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if len(e.ValidationErrors) > 0 {
		var parts []string
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(e.ValidationErrors))
		for k := range e.ValidationErrors {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, field := range keys {
			for _, msg := range e.ValidationErrors[field] {
				parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
			}
		}
		return "Validation failed: " + strings.Join(parts, "; ")
	}
	return e.Message
}

// exitCodeForStatus maps an HTTP status code to a CLI exit code.
func exitCodeForStatus(status int) int {
	switch {
	case status == 401 || status == 403:
		return 2
	case status == 422:
		return 3
	case status == 404:
		return 4
	case status >= 500:
		return 5
	default:
		return 1
	}
}

// standardResponse represents the standard API response format:
// {"data": {}, "message": "...", "http_status": N}
type standardResponse struct {
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status"`
}

// validationResponse represents the validation error format:
// {"errors": {"field": ["msg"]}}
type validationResponse struct {
	Errors map[string][]string `json:"errors"`
}

// ParseErrorResponse reads an HTTP response body and parses it into an APIError.
// It handles both standard and validation API response formats, with a fallback
// for malformed JSON.
func ParseErrorResponse(resp *http.Response) *APIError {
	status := resp.StatusCode
	exitCode := exitCodeForStatus(status)

	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return &APIError{
			Message:    http.StatusText(status),
			HTTPStatus: status,
			ExitCode:   exitCode,
		}
	}

	// Try validation response format first ({"errors": {"field": ["msg"]}}).
	var valResp validationResponse
	if json.Unmarshal(body, &valResp) == nil && len(valResp.Errors) > 0 {
		apiErr := &APIError{
			HTTPStatus:       status,
			ExitCode:         exitCode,
			ValidationErrors: valResp.Errors,
		}
		// Set the message via Error() so it's populated.
		apiErr.Message = apiErr.Error()
		return apiErr
	}

	// Try standard response format ({"data": {}, "message": "...", "http_status": N}).
	var stdResp standardResponse
	if json.Unmarshal(body, &stdResp) == nil && stdResp.Message != "" {
		return &APIError{
			Message:    stdResp.Message,
			HTTPStatus: status,
			ExitCode:   exitCode,
		}
	}

	// Fallback for malformed JSON or unexpected format.
	return &APIError{
		Message:    http.StatusText(status),
		HTTPStatus: status,
		ExitCode:   exitCode,
	}
}
