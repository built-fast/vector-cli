package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_DefaultUserAgent(t *testing.T) {
	c := NewClient("https://api.example.com", "tok", "")
	assert.Contains(t, c.UserAgent, "vector-cli/")
}

func TestNewClient_CustomUserAgent(t *testing.T) {
	c := NewClient("https://api.example.com", "tok", "custom/1.0")
	assert.Equal(t, "custom/1.0", c.UserAgent)
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com/", "tok", "")
	assert.Equal(t, "https://api.example.com", c.BaseURL)
}

func TestClient_HeaderInjection(t *testing.T) {
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token", "vector-cli/test")
	_, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)

	assert.Equal(t, "Bearer test-token", gotHeaders.Get("Authorization"))
	assert.Equal(t, "application/json", gotHeaders.Get("Accept"))
	assert.Equal(t, "vector-cli/test", gotHeaders.Get("User-Agent"))
}

func TestClient_NoAuthHeaderWithoutToken(t *testing.T) {
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "vector-cli/test")
	_, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)

	assert.Empty(t, gotHeaders.Get("Authorization"))
}

func TestClient_Get(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	query := url.Values{"page": []string{"1"}, "limit": []string{"10"}}
	resp, err := c.Get(context.Background(), "/api/v1/items", query)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/api/v1/items", gotPath)
	assert.Contains(t, gotQuery, "page=1")
	assert.Contains(t, gotQuery, "limit=10")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_GetWithoutQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	resp, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, gotQuery)
}

func TestClient_Post(t *testing.T) {
	var gotMethod, gotContentType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":1}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	body := map[string]string{"name": "test", "email": "test@example.com"}
	resp, err := c.Post(context.Background(), "/api/v1/items", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "test", gotBody["name"])
	assert.Equal(t, "test@example.com", gotBody["email"])
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestClient_Put(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":1}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	body := map[string]string{"name": "updated"}
	resp, err := c.Put(context.Background(), "/api/v1/items/1", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "updated", gotBody["name"])
}

func TestClient_Delete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	resp, err := c.Delete(context.Background(), "/api/v1/items/1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/api/v1/items/1", gotPath)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestClient_PutFile(t *testing.T) {
	var gotMethod, gotUserAgent string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUserAgent = r.Header.Get("User-Agent")
		gotBody, _ = io.ReadAll(r.Body)
		// PutFile should not add Authorization or Accept headers (presigned S3 URL).
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("file-content"), 0644))

	f, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	c := NewClient("https://api.example.com", "tok", "vector-cli/test")
	// PutFile uses the full URL directly (presigned S3 URL), not BaseURL+path.
	resp, err := c.PutFile(context.Background(), srv.URL+"/upload", f)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "vector-cli/test", gotUserAgent)
	assert.Equal(t, "file-content", string(gotBody))
}

func TestClient_PutFilePart(t *testing.T) {
	var gotMethod, gotUserAgent string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUserAgent = r.Header.Get("User-Agent")
		gotBody, _ = io.ReadAll(r.Body)
		// PutFilePart should not add Authorization or Accept headers (presigned S3 URL).
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Accept"))
		w.Header().Set("Etag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	content := []byte("chunk-data-here")
	reader := bytes.NewReader(content)

	c := NewClient("https://api.example.com", "tok", "vector-cli/test")
	etag, err := c.PutFilePart(context.Background(), srv.URL+"/upload/part1", reader, int64(len(content)))
	require.NoError(t, err)

	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "vector-cli/test", gotUserAgent)
	assert.Equal(t, "chunk-data-here", string(gotBody))
	assert.Equal(t, `"abc123"`, etag)
}

func TestClient_PutFilePartErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	content := []byte("data")
	reader := bytes.NewReader(content)

	c := NewClient("https://api.example.com", "tok", "")
	_, err := c.PutFilePart(context.Background(), srv.URL+"/upload", reader, int64(len(content)))
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, 403, apiErr.HTTPStatus)
}

func TestClient_PutFilePartMissingEtag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 200 but no ETag header
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	content := []byte("data")
	reader := bytes.NewReader(content)

	c := NewClient("https://api.example.com", "tok", "")
	_, err := c.PutFilePart(context.Background(), srv.URL+"/upload", reader, int64(len(content)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing ETag")
}

func TestClient_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"data":{},"message":"Unauthenticated.","http_status":401}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-token", "")
	_, err := c.Get(context.Background(), "/api/v1/ping", nil)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, 401, apiErr.HTTPStatus)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Equal(t, "Unauthenticated.", apiErr.Message)
}

func TestClient_ValidationErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":{"name":["The name field is required."]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	_, err := c.Post(context.Background(), "/api/v1/items", map[string]string{})
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, 422, apiErr.HTTPStatus)
	assert.Equal(t, 3, apiErr.ExitCode)
	assert.Contains(t, apiErr.Error(), "name: The name field is required.")
}

func TestClient_ServerErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "")
	_, err := c.Delete(context.Background(), "/api/v1/items/1")
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, 500, apiErr.HTTPStatus)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestClient_PutFileErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0644))

	f, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	c := NewClient("https://api.example.com", "tok", "")
	_, err = c.PutFile(context.Background(), srv.URL+"/upload", f)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "error should be *APIError")
	assert.Equal(t, 403, apiErr.HTTPStatus)
}
