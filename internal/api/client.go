package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/built-fast/vector-cli/internal/version"
)

// Client is an HTTP client for the Vector API.
type Client struct {
	BaseURL    string
	Token      string
	UserAgent  string
	httpClient *http.Client
}

// NewClient creates a new API client. If userAgent is empty, it defaults to
// "vector-cli/<version>".
func NewClient(baseURL, token, userAgent string) *Client {
	if userAgent == "" {
		userAgent = "vector-cli/" + version.Version
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		UserAgent:  userAgent,
		httpClient: &http.Client{},
	}
}

// Get performs a GET request to the given API path with optional query parameters.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	reqURL := c.BaseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}
	return c.do(req)
}

// Post performs a POST request with a JSON-encoded body.
func (c *Client) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.jsonRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON-encoded body.
func (c *Client) Put(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.jsonRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request to the given API path.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating DELETE request: %w", err)
	}
	return c.do(req)
}

// PutFile uploads a file via PUT to the given URL (typically a presigned S3 URL).
// Unlike other methods, this sends the raw file content and does not add
// Authorization or Accept headers.
func (c *Client) PutFile(ctx context.Context, url string, file *os.File) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, file)
	if err != nil {
		return nil, fmt.Errorf("creating file upload request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing file upload: %w", err)
	}
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, ParseErrorResponse(resp)
	}
	return resp, nil
}

// PutFilePart uploads a file part via PUT to the given URL (typically a presigned
// S3 URL for multipart uploads). It returns the ETag header from the response.
// Unlike other methods, this does not add Authorization or Accept headers.
func (c *Client) PutFilePart(ctx context.Context, url string, body io.Reader, contentLength int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return "", fmt.Errorf("creating file part upload request: %w", err)
	}
	req.ContentLength = contentLength
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing file part upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", ParseErrorResponse(resp)
	}

	etag := resp.Header.Get("Etag")
	if etag == "" {
		return "", fmt.Errorf("S3 response missing ETag header")
	}
	return etag, nil
}

// jsonRequest is a helper that JSON-encodes a body and sends a request.
// When body is nil, the request is sent with no body and no Content-Type header.
func (c *Client) jsonRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	hasBody := body != nil
	if hasBody {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
	}
	var req *http.Request
	var err error
	if hasBody {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("creating %s request: %w", method, err)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req)
}

// do executes a request, adding standard headers and handling error responses.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseErrorResponse(resp)
	}
	return resp, nil
}
