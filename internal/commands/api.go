package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/output"
)

const apiBasePath = "/api/v1/vector/"

// apiStdinReader is the source for fields and bodies read from stdin (the "@-"
// and "--input -" forms). It is a variable so tests can override it, mirroring
// confirmReader and stdinReader elsewhere in this package.
var apiStdinReader io.Reader = os.Stdin

// methodsWithBody are the HTTP methods that carry collected fields as a JSON
// request body. Any other method appends them to the URL query string instead.
var methodsWithBody = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// NewAPICmd creates the api passthrough command.
func NewAPICmd() *cobra.Command {
	var (
		method     string
		rawFields  []string
		fields     []string
		input      string
		reqHeaders []string
		include    bool
		verbose    bool
		paginate   bool
	)

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated request to the Vector Pro API",
		Long: "Send an authenticated HTTP request to any Vector Pro API endpoint and " +
			"print the raw response.\n\n" +
			"An <endpoint> beginning with \"/\" is sent verbatim against the base URL. " +
			"Any other value has \"/api/v1/vector/\" prepended, so \"sites\" resolves to " +
			"\"/api/v1/vector/sites\".\n\n" +
			"The method defaults to GET, or POST when any field or --input is given. " +
			"Fields supplied with -f/-F are encoded as a JSON body for " +
			"POST/PUT/PATCH/DELETE, or as query parameters for GET/HEAD. Use the " +
			"key[]=value suffix to build arrays; reusing a plain key is an error.\n\n" +
			"Use -i/--include to print the response status line and headers before " +
			"the body, and --verbose to echo the resolved request to stderr before " +
			"sending it.",
		Example: `  # GET a resource that has no dedicated subcommand
  vector api php-versions

  # Equivalent to the line above, with an absolute path
  vector api /api/v1/vector/php-versions

  # Filter the response with built-in jq
  vector api sites --jq '.data[].id'

  # Create a resource with typed fields (auto-POST)
  vector api sites -f customer_id=cust_123 -F dev_php_version=8.3

  # Send a raw request body from a file or stdin
  vector api sites --method POST --input body.json
  echo '{"customer_id":"cust_123"}' | vector api sites -X POST --input -

  # Send a custom request header
  vector api sites -H 'Accept: text/plain'

  # Inspect the response status line and headers
  vector api sites -i

  # Echo the resolved request to stderr before sending
  vector api sites --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: apiRunE(&method, &rawFields, &fields, &input, &reqHeaders, &include, &verbose, &paginate),
	}

	cmd.Flags().StringVarP(&method, "method", "X", http.MethodGet,
		"HTTP method (GET, POST, PUT, PATCH, DELETE)")
	cmd.Flags().StringArrayVarP(&rawFields, "raw-field", "f", nil,
		"add a string parameter in key=value format (repeatable)")
	cmd.Flags().StringArrayVarP(&fields, "field", "F", nil,
		"add a typed parameter in key=value format; @file/@- load the value (repeatable)")
	cmd.Flags().StringVar(&input, "input", "",
		"send a raw request body read from a file, or from stdin when set to -")
	cmd.Flags().StringArrayVarP(&reqHeaders, "header", "H", nil,
		"add a request header in key:value format; overrides defaults (repeatable)")
	cmd.Flags().BoolVarP(&include, "include", "i", false,
		"print the response status line and headers before the body")
	cmd.Flags().BoolVar(&verbose, "verbose", false,
		"echo the resolved request (method, URL, body) to stderr before sending")
	cmd.Flags().BoolVar(&paginate, "paginate", false,
		"fetch every page of a paginated collection, merging results into one JSON array")

	return cmd
}

// apiRunE returns the RunE for the api passthrough command.
func apiRunE(method *string, rawFields, fields *[]string, input *string, reqHeaders *[]string, include, verbose, paginate *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		app, err := requireApp(cmd)
		if err != nil {
			return err
		}

		customHeaders, err := parseHeaders(*reqHeaders)
		if err != nil {
			return err
		}

		hasFields := len(*rawFields) > 0 || len(*fields) > 0
		hasInput := cmd.Flags().Changed("input")

		if hasInput && hasFields {
			return &api.APIError{
				Message:  "--input cannot be combined with -f/--raw-field or -F/--field",
				ExitCode: 3,
			}
		}

		if *paginate && *include {
			return &api.APIError{
				Message:  "--paginate cannot be combined with -i/--include",
				ExitCode: 3,
			}
		}

		// Default to POST when a body source is supplied without an explicit method.
		resolvedMethod := strings.ToUpper(*method)
		if !cmd.Flags().Changed("method") && (hasFields || hasInput) {
			resolvedMethod = http.MethodPost
		}

		path := resolveAPIPath(args[0])

		var (
			bodyBytes []byte
			headers   http.Header
		)

		switch {
		case hasInput:
			bodyBytes, err = readInputBody(*input)
			if err != nil {
				return err
			}
			headers = jsonContentTypeHeader()

		case hasFields:
			collected, err := collectFields(*rawFields, *fields)
			if err != nil {
				return err
			}

			if methodsWithBody[resolvedMethod] {
				bodyBytes, err = json.Marshal(collected)
				if err != nil {
					return fmt.Errorf("failed to encode request body: %w", err)
				}
				headers = jsonContentTypeHeader()
			} else {
				path = appendQueryParams(path, collected)
			}
		}

		// Custom -H headers override any default we set above (e.g. the JSON
		// Content-Type) as well as the client's default Authorization/Accept.
		headers = mergeHeaders(headers, customHeaders)

		if *verbose {
			writeVerboseRequest(cmd.ErrOrStderr(), resolvedMethod, app.Client.BaseURL+path, bodyBytes)
		}

		if *paginate {
			return runAPIPaginate(cmd, app, resolvedMethod, path, headers, bodyBytes)
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		resp, err := app.Client.Do(cmd.Context(), resolvedMethod, path, headers, reqBody)
		if err != nil {
			return fmt.Errorf("failed to make API request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to make API request: %w", err)
		}

		if *include {
			writeResponseHead(app.Output.Underlying(), resp)
		}

		return writeAPIResponse(app.Output, body)
	}
}

// resolveAPIPath maps an endpoint argument to a request path. A value starting
// with "/" is returned verbatim; any other value is appended to the Vector Pro
// base path "/api/v1/vector/".
func resolveAPIPath(endpoint string) string {
	if strings.HasPrefix(endpoint, "/") {
		return endpoint
	}
	return apiBasePath + endpoint
}

// jsonContentTypeHeader returns a header set declaring a JSON request body.
func jsonContentTypeHeader() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	return h
}

// parseHeaders converts -H "key:value" specs into an http.Header. The value is
// trimmed of leading whitespace after the colon, matching gh api. A spec without
// a colon is a loud client-side error (exit code 3). A nil/empty input returns a
// nil header so no custom headers are sent.
func parseHeaders(specs []string) (http.Header, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	headers := http.Header{}
	for _, spec := range specs {
		key, value, ok := strings.Cut(spec, ":")
		if !ok || key == "" {
			return nil, &api.APIError{
				Message:  fmt.Sprintf("invalid header %q: expected key:value", spec),
				ExitCode: 3,
			}
		}
		headers.Add(key, strings.TrimSpace(value))
	}
	return headers, nil
}

// mergeHeaders layers custom headers over base headers. A custom header replaces
// any same-named base header so an explicit -H wins over a default we set. Either
// argument may be nil.
func mergeHeaders(base, custom http.Header) http.Header {
	if len(custom) == 0 {
		return base
	}

	merged := http.Header{}
	for key, values := range base {
		merged[key] = values
	}
	for key, values := range custom {
		merged[http.CanonicalHeaderKey(key)] = values
	}
	return merged
}

// collectFields merges -f (raw string) and -F (typed) fields into a single
// ordered map suitable for JSON or query encoding. Raw fields are always
// strings; typed fields coerce true/false/null and numeric literals to JSON
// types and load @file/@- values. A key without the "[]" suffix may appear only
// once across all fields; reusing it is a loud error (exit code 3). Keys with
// the "[]" suffix accumulate into an array.
func collectFields(rawFields, fields []string) (map[string]any, error) {
	collected := map[string]any{}
	// scalarKeys tracks plain (non-array) keys that have been set, so a reuse
	// can be rejected.
	scalarKeys := map[string]bool{}
	arrayKeys := map[string]bool{}

	add := func(spec string, typed bool) error {
		key, rawValue, ok := strings.Cut(spec, "=")
		if !ok || key == "" {
			return &api.APIError{
				Message:  fmt.Sprintf("invalid field %q: expected key=value", spec),
				ExitCode: 3,
			}
		}

		isArray := strings.HasSuffix(key, "[]")
		name := strings.TrimSuffix(key, "[]")

		var value any = rawValue
		if typed {
			coerced, err := coerceFieldValue(rawValue)
			if err != nil {
				return err
			}
			value = coerced
		}

		if isArray {
			if scalarKeys[name] {
				return reusedKeyError(name)
			}
			arr, _ := collected[name].([]any)
			collected[name] = append(arr, value)
			arrayKeys[name] = true
			return nil
		}

		if scalarKeys[name] || arrayKeys[name] {
			return reusedKeyError(name)
		}
		collected[name] = value
		scalarKeys[name] = true
		return nil
	}

	for _, spec := range rawFields {
		if err := add(spec, false); err != nil {
			return nil, err
		}
	}
	for _, spec := range fields {
		if err := add(spec, true); err != nil {
			return nil, err
		}
	}

	return collected, nil
}

// reusedKeyError reports a key used more than once without the "[]" suffix.
func reusedKeyError(name string) error {
	return &api.APIError{
		Message:  fmt.Sprintf("unexpected override existing field under %q", name),
		ExitCode: 3,
	}
}

// coerceFieldValue maps a -F value to a JSON type. true/false/null become their
// JSON equivalents, integer and float literals become numbers, an @filename
// loads the value from a file, @- reads it from stdin, and everything else is
// kept as a string.
func coerceFieldValue(value string) (any, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}

	if strings.HasPrefix(value, "@") {
		raw, err := readFileOrStdin(value[1:])
		if err != nil {
			return nil, err
		}
		return string(raw), nil
	}

	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, nil
	}

	return value, nil
}

// appendQueryParams encodes collected fields onto a path's query string. Array
// values are emitted as repeated key entries; scalars are stringified.
func appendQueryParams(path string, collected map[string]any) string {
	query := url.Values{}
	for key, value := range collected {
		if arr, ok := value.([]any); ok {
			for _, item := range arr {
				query.Add(key, fmt.Sprint(item))
			}
			continue
		}
		query.Set(key, fmt.Sprint(value))
	}

	if len(query) == 0 {
		return path
	}

	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + query.Encode()
}

// readInputBody resolves the --input value into raw request body bytes. A value
// of "-" reads from stdin; anything else is treated as a file path.
func readInputBody(input string) ([]byte, error) {
	return readFileOrStdin(input)
}

// readFileOrStdin reads from stdin when source is "-", otherwise from the file
// at the given path.
func readFileOrStdin(source string) ([]byte, error) {
	if source == "-" {
		raw, err := io.ReadAll(apiStdinReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		return raw, nil
	}

	raw, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return raw, nil
}

// writeVerboseRequest echoes the resolved request (method, URL, and body when
// present) to the given writer, which is always stderr so stdout stays pipeable.
func writeVerboseRequest(w io.Writer, method, url string, body []byte) {
	fmt.Fprintf(w, "> %s %s\n", method, url)
	if len(body) > 0 {
		fmt.Fprintf(w, "> %s\n", body)
	}
}

// writeResponseHead prints the HTTP status line and response headers, sorted by
// name, followed by a blank line. It runs before the body when -i/--include is
// set.
func writeResponseHead(w io.Writer, resp *http.Response) {
	fmt.Fprintf(w, "%s %s\n", resp.Proto, resp.Status)

	names := make([]string, 0, len(resp.Header))
	for name := range resp.Header {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		for _, value := range resp.Header[name] {
			fmt.Fprintf(w, "%s: %s\n", name, value)
		}
	}
	fmt.Fprintln(w)
}

// writeAPIResponse prints the response body. When the body parses as JSON it is
// pretty-printed (and the --jq filter, if set, is applied); otherwise the raw
// bytes are written verbatim.
func writeAPIResponse(w *output.Writer, body []byte) error {
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		return w.JSON(parsed)
	}

	_, err := w.Underlying().Write(body)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}

// maxAPIPages caps how many pages --paginate fetches, guarding against an
// unbounded loop when a server never reports a final page. Server rate limits
// are the backstop for legitimately larger result sets.
const maxAPIPages = 100

// apiPageEnvelope is the {data, meta} shape that --paginate follows. A response
// that does not match this shape is treated as a single, non-paginated result.
type apiPageEnvelope struct {
	Data json.RawMessage `json:"data"`
	Meta *struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
	} `json:"meta"`
}

// runAPIPaginate follows pagination (meta.current_page / meta.last_page),
// incrementing the page query parameter until the last page is reached, and
// emits every page's data array merged into a single JSON array. A response
// without the {data, meta} shape yields a single request returned unchanged.
// Fetching stops after maxAPIPages with a warning to stderr.
func runAPIPaginate(cmd *cobra.Command, app *appctx.App, method, path string, headers http.Header, bodyBytes []byte) error {
	merged := []json.RawMessage{}

	for page := 1; ; page++ {
		pagePath, err := withPageParam(path, page)
		if err != nil {
			return err
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		resp, err := app.Client.Do(cmd.Context(), method, pagePath, headers, reqBody)
		if err != nil {
			return fmt.Errorf("failed to make API request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to make API request: %w", err)
		}

		var envelope apiPageEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil || envelope.Meta == nil {
			// Not a paginated {data, meta} collection: return the first response
			// unchanged. A later page lacking the shape just stops the loop.
			if page == 1 {
				return writeAPIResponse(app.Output, body)
			}
			break
		}

		if len(envelope.Data) > 0 {
			items, err := pageItems(envelope.Data)
			if err != nil {
				return err
			}
			merged = append(merged, items...)
		}

		if envelope.Meta.CurrentPage >= envelope.Meta.LastPage {
			break
		}

		if page >= maxAPIPages {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"warning: --paginate stopped after %d pages before reaching the last page\n", maxAPIPages)
			break
		}
	}

	return app.Output.JSON(merged)
}

// withPageParam returns path with its "page" query parameter set to page,
// preserving any other query parameters already present.
func withPageParam(path string, page int) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", &api.APIError{
			Message:  fmt.Sprintf("invalid path %q: %v", path, err),
			ExitCode: 3,
		}
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// pageItems decodes a page's data array into individual JSON elements.
func pageItems(data json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse paginated response: %w", err)
	}
	return items, nil
}
