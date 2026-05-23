package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error_SimpleMessage(t *testing.T) {
	err := &APIError{Message: "Something went wrong"}
	assert.Equal(t, "Something went wrong", err.Error())
}

func TestAPIError_Error_ValidationErrors(t *testing.T) {
	err := &APIError{
		Message: "ignored when validation errors present",
		ValidationErrors: map[string][]string{
			"your_customer_id": {"The partner customer id field is required."},
		},
	}
	assert.Equal(t, "Validation failed: your_customer_id: The partner customer id field is required.", err.Error())
}

func TestAPIError_Error_MultipleValidationErrors(t *testing.T) {
	err := &APIError{
		ValidationErrors: map[string][]string{
			"email": {"The email field is required.", "The email must be valid."},
			"name":  {"The name field is required."},
		},
	}
	result := err.Error()
	assert.Contains(t, result, "Validation failed: ")
	assert.Contains(t, result, "email: The email field is required.")
	assert.Contains(t, result, "email: The email must be valid.")
	assert.Contains(t, result, "name: The name field is required.")
}

func TestAPIError_ImplementsErrorInterface(t *testing.T) {
	var err error = &APIError{Message: "test"}
	require.Error(t, err)
	assert.Equal(t, "test", err.Error())
}

func TestExitCodeForStatus(t *testing.T) {
	tests := []struct {
		status   int
		exitCode int
	}{
		{401, 2},
		{403, 2},
		{404, 4},
		{422, 3},
		{500, 5},
		{502, 5},
		{503, 5},
		{400, 1},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.exitCode, exitCodeForStatus(tt.status))
		})
	}
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestParseErrorResponse_StandardFormat(t *testing.T) {
	resp := newResponse(401, `{"data": {}, "message": "Unauthenticated.", "http_status": 401}`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Unauthenticated.", apiErr.Message)
	assert.Equal(t, 401, apiErr.HTTPStatus)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestParseErrorResponse_ValidationFormat(t *testing.T) {
	body := `{"errors": {"your_customer_id": ["The partner customer id field is required."]}}`
	resp := newResponse(422, body)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, 422, apiErr.HTTPStatus)
	assert.Equal(t, 3, apiErr.ExitCode)
	assert.Contains(t, apiErr.ValidationErrors, "your_customer_id")
	assert.Equal(t, "Validation failed: your_customer_id: The partner customer id field is required.", apiErr.Error())
}

func TestParseErrorResponse_MalformedJSON(t *testing.T) {
	resp := newResponse(500, `not json at all`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Internal Server Error", apiErr.Message)
	assert.Equal(t, 500, apiErr.HTTPStatus)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestParseErrorResponse_EmptyBody(t *testing.T) {
	resp := newResponse(404, "")
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Not Found", apiErr.Message)
	assert.Equal(t, 404, apiErr.HTTPStatus)
	assert.Equal(t, 4, apiErr.ExitCode)
}

func TestParseErrorResponse_403Forbidden(t *testing.T) {
	resp := newResponse(403, `{"data": {}, "message": "Forbidden.", "http_status": 403}`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Forbidden.", apiErr.Message)
	assert.Equal(t, 403, apiErr.HTTPStatus)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestParseErrorResponse_404NotFound(t *testing.T) {
	resp := newResponse(404, `{"data": {}, "message": "Resource not found.", "http_status": 404}`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Resource not found.", apiErr.Message)
	assert.Equal(t, 404, apiErr.HTTPStatus)
	assert.Equal(t, 4, apiErr.ExitCode)
}

func TestParseErrorResponse_5xxServerError(t *testing.T) {
	resp := newResponse(503, `{"data": {}, "message": "Service unavailable.", "http_status": 503}`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Service unavailable.", apiErr.Message)
	assert.Equal(t, 503, apiErr.HTTPStatus)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestParseErrorResponse_EmptyJSONObject(t *testing.T) {
	resp := newResponse(500, `{}`)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Equal(t, "Internal Server Error", apiErr.Message)
	assert.Equal(t, 500, apiErr.HTTPStatus)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestParseErrorResponse_MultipleValidationFields(t *testing.T) {
	body := `{"errors": {"email": ["The email field is required."], "name": ["The name field is required."]}}`
	resp := newResponse(422, body)
	apiErr := ParseErrorResponse(resp)

	require.NotNil(t, apiErr)
	assert.Len(t, apiErr.ValidationErrors, 2)
	assert.Equal(t, 3, apiErr.ExitCode)
}
