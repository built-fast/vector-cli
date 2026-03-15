# Vector CLI Style Guide

Conventions for contributors and agents working on vector-cli.

## Command Constructors

Exported `NewXxxCmd() *cobra.Command` for top-level command groups and nested
subgroups that are referenced from another file (e.g., `NewSiteSSHKeyCmd()` is
called from `site.go`). Unexported `newXxxCmd()` for leaf commands that live in
the same file. The rule: exported = referenced cross-file, unexported = same file
only.

Group constructors build the command, call `cmd.AddCommand(...)` for each
subcommand, and return. They never set `RunE`.

Leaf constructors build the command, set `RunE` (inline closure or factory
function), register local flags, and return.

## RunE Body Sequence

Leaf commands follow a canonical sequence. Not every step applies to every
command, but the order is fixed:

1. `requireApp(cmd)` — extract App from context, verify auth token
2. Read flags / build request body
3. Confirmation prompt (destructive operations only)
4. API call (`app.Client.Get/Post/Put/Delete`)
5. `defer func() { _ = resp.Body.Close() }()`
6. `io.ReadAll(resp.Body)`
7. `parseResponseData(body)` or `parseResponseWithMeta(body)`
8. JSON branch — `if app.Output.Format() == output.JSON { return app.Output.JSON(...) }`
9. `json.Unmarshal` into `map[string]any` or `[]map[string]any`
10. Format and output (`app.Output.Table`, `app.Output.KeyValue`, `app.Output.Message`)
11. `return nil`

For repetitive site actions, use the `siteActionRunE` and `sitePostActionRunE`
factory functions instead of duplicating the sequence.

## Output

Format detection follows `--json` > `--no-json` > TTY auto-detect (TTY → Table,
piped → JSON). The `--jq` flag forces JSON mode and is mutually exclusive with
`--no-json`.

Writer methods for formatted output:

- `app.Output.JSON(v)` — pretty-printed JSON (applies jq filter if set)
- `app.Output.Table(headers, rows)` — tabular output via tabwriter
- `app.Output.KeyValue(pairs)` — right-aligned key: value pairs (show commands)
- `app.Output.Pagination(page, lastPage, total)` — "Page X of Y (Z total)"
- `app.Output.Message(msg)` — plain text line

Standalone `Print*` helpers (`PrintTable`, `PrintJSON`, `PrintKeyValue`,
`PrintPagination`, `PrintMessage`, `PrintError`) are used when no Writer is
available (e.g., `printLogEntries` writing to `cmd.OutOrStdout()`).

## Error Handling

Wrap errors with `fmt.Errorf("failed to <verb> <noun>: %w", err)`. Use a
consistent action phrase per command (e.g., "failed to list sites",
"failed to create site").

Exit codes are mapped from HTTP status in `api.exitCodeForStatus`:

| HTTP Status | Exit Code | Meaning |
|-------------|-----------|---------|
| 401, 403 | 2 | Authentication/authorization |
| 422 | 3 | Validation error |
| 404 | 4 | Not found |
| 5xx | 5 | Server error |
| other | 1 | General error |

Return `*api.APIError` directly for client-side validation failures (e.g.,
missing required flag), setting `ExitCode` to match the equivalent HTTP status
category.

## Flags

All leaf command flags are **local** (`cmd.Flags()`). Persistent flags live only
on the root command: `--token`, `--json`, `--no-json`, `--jq`.

Flag names use **kebab-case** (`--customer-id`, `--php-version`, `--cache-tag`).

Use `cmd.Flags().Changed("flag-name")` to distinguish "flag not passed" from
"flag passed with zero value" — required for optional PATCH/PUT fields that
should only be included in the request body when explicitly set.

## Config Resolution

Token precedence: `--token` flag > `VECTOR_API_KEY` env > OS keyring.

Config directory: `VECTOR_CONFIG_DIR` env > `XDG_CONFIG_HOME/vector` >
`~/.config/vector` (Linux/macOS) or `%APPDATA%/vector` (Windows).

Keyring disabled via `VECTOR_NO_KEYRING` env (any non-empty value).

## Bare Command Groups

Group commands (resource nouns like `site`, `env`, `waf`, `backup`) must **not**
set `RunE`. Bare invocation shows help automatically via Cobra.

The root command is the only exception — it sets `RunE` to handle `--version`.

## File Organization

One file per command group in `internal/commands/`. Nested subgroups get their
own file named with underscores: `site_ssh_key.go`, `waf_blocked_ip.go`,
`env_secret.go`.

Tests mirror source files: `site_test.go`, `waf_blocked_ip_test.go`.

Shared helpers live in `helpers.go` (`requireApp`, pagination, parsing,
formatting, confirm).

## API Base Paths

Declare a package-level `const` at the top of each file:

```go
const sitesBasePath = "/api/v1/vector/sites"
```

For nested resource paths, use helper functions:

```go
func wafBlockedIPsPath(siteID string) string {
    return sitesBasePath + "/" + siteID + "/waf/blocked-ips"
}
```

## Import Ordering

Three groups separated by blank lines, each alphabetically sorted:

1. Standard library
2. Third-party modules
3. Project-internal (`github.com/built-fast/vector-cli/...`)

`goimports` enforces this.

## Declaration Ordering

Within a command file, declarations follow this order:

1. Package-level constants (base paths)
2. Package-level variables (if any)
3. Exported group constructor (`NewXxxCmd`)
4. Unexported leaf constructors (in the same order as `AddCommand` calls)
5. Private helper functions

## Testing

Same-package tests (`package commands`). Test files follow these patterns:

**Command builder**: `buildXxxCmd(baseURL, token, format)` returns
`(*cobra.Command, *bytes.Buffer, *bytes.Buffer)` — root command wired with App
context, stdout buffer, stderr buffer.

**Test server**: `newXxxTestServer(validToken)` returns `*httptest.Server` with
a `method + path` switch dispatching fixture responses.

**Fixture variables**: Package-level `var xxxResponse = map[string]any{...}` for
each endpoint response.

**Confirm override**: Replace `confirmReader` with `strings.NewReader(...)` and
restore via `t.Cleanup`.

**Keyring**: Call `keyring.MockInit()` per test (not TestMain). Use
`t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())` to isolate config.

**Assertions**: `require` for preconditions and fatal checks, `assert` for
outcome verification. Prefer `require.NoError` / `require.Error` for error
checks that guard subsequent assertions.

**E2E tests**: BATS scripts in `e2e/` using a Prism mock server against the
OpenAPI spec (`e2e/openapi.yaml`).
