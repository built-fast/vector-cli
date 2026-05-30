---
name: vector
description: Reference document for the vector CLI — manages sites, environments, deployments, backups, WAF, SSL, and more on the Vector Pro hosting platform.
triggers:
  - vector
---

# Vector CLI — Agent Reference

`vector` is the CLI for managing sites on the **Vector Pro** hosting platform
(API: `https://api.builtfast.com`). This document is the authoritative reference
for AI agents invoking vector commands.

## Agent Invariants

### Output Modes

| Priority | Mechanism | Result |
|----------|-----------|--------|
| 1 | `--json` flag | JSON output |
| 2 | `--no-json` flag | Table output |
| 3 | `--jq <expr>` flag | JSON output with jq filter applied |
| 4 | TTY auto-detect | TTY → table, piped → JSON |

- `--jq` implies `--json` and is mutually exclusive with `--no-json`.
- Agents should always pass `--json` for machine-readable output.
- Use `--jq '.data'` or similar to extract specific fields.

### Authentication

Token resolution order:

1. `--token <value>` flag (highest priority)
2. `VECTOR_API_KEY` environment variable
3. OS keyring (stored via `vector auth login`)

All commands except `vector auth login` require a valid token.

### Exit Codes

| Code | Meaning | HTTP Status |
|------|---------|-------------|
| 0 | Success | 2xx |
| 1 | General error | other |
| 2 | Auth failure | 401, 403 |
| 3 | Validation error | 422 |
| 4 | Not found | 404 |
| 5 | Server error | 5xx |

### Pagination

List commands accept `--page` (default 1) and `--per-page` (default 15).
JSON output includes `meta.current_page`, `meta.last_page`, and `meta.total`.

### Destructive Operations

Commands that delete or suspend resources require interactive confirmation
unless `--force` is passed.

### Waiting for Async Operations

Four commands support `--wait` to block until the operation reaches a terminal
status instead of returning immediately:

| Command | Terminal Status | Failed Statuses |
|---------|----------------|-----------------|
| `site create` | `active` | `failed` |
| `deploy trigger` | `deployed` | `failed`, `cancelled` |
| `deploy rollback` | `deployed` | `failed`, `cancelled` |
| `restore create` | `completed` | `failed` |

**Shared flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--wait` | false | Enable blocking wait |
| `--poll-interval` | 60s | Poll frequency (min 1s, must be ≤ timeout) |
| `--timeout` | 5m | Maximum wait time (max 30m) |

**Behavior:**
- TTY: displays a live-updating alternate screen with status, then prints
  a summary line and final state on exit.
- JSON mode (`--json`): silently polls and emits only the final JSON object.
- Piped/non-TTY: silently polls with no ANSI output.
- Returns exit code 1 if the operation reaches a failed status or times out.
- Ctrl+C cleanly aborts the wait.
- `site create --wait` prints one-time credentials (SFTP, DB, WP admin)
  before entering the wait loop; with `--json`, credentials are merged into
  the final JSON output.

**Agents** should prefer `--wait --json` to get a single blocking call that
returns the final resource state, eliminating the need for manual poll loops.

---

## Authentication

### vector auth login

Authenticate and store token in the OS keyring.

```
vector auth login
```

Prompts for a token interactively, validates it via the API, and stores it.
Exits with code 2 if the token is invalid.

### vector auth logout

Remove stored credentials from the keyring.

```
vector auth logout
```

### vector auth status

Show current authentication status.

```
vector auth status --json
```

Displays: user name/email, account name, token name, abilities, expiration,
token source (flag/env/keyring), and config directory.
Exits with code 2 if not authenticated.

---

## Configuration

| Setting | Source |
|---------|--------|
| Config directory | `VECTOR_CONFIG_DIR` > `XDG_CONFIG_HOME/vector` > `~/.config/vector` |
| Disable keyring | `VECTOR_NO_KEYRING=1` |
| API token | `--token` > `VECTOR_API_KEY` > keyring |

---

## Command Reference

### Sites

#### vector site list

```
vector site list [--page N] [--per-page N] [--json]
```

Lists all sites. Columns: ID, CUSTOMER ID, STATUS, DEV DOMAIN, TAGS.

#### vector site show

```
vector site show <site-id> [--json]
```

Displays site details including environments table.

#### vector site create

```
vector site create --customer-id <id> [--php-version <ver>] [--tags <t1,t2>] \
  [--production-domain <domain>] [--staging-domain <domain>] \
  [--wp-admin-email <email>] [--wp-admin-user <user>] [--wp-site-title <title>] \
  [--wait] [--poll-interval <duration>] [--timeout <duration>]
```

Creates a new site. Returns SFTP, DB, and WordPress credentials (shown once).

| Flag | Required | Description |
|------|----------|-------------|
| `--customer-id` | yes | Customer identifier |
| `--php-version` | no | PHP version |
| `--tags` | no | Comma-separated tags |
| `--production-domain` | no | Production domain |
| `--staging-domain` | no | Staging domain |
| `--wp-admin-email` | no | WordPress admin email |
| `--wp-admin-user` | no | WordPress admin username |
| `--wp-site-title` | no | WordPress site title |
| `--wait` | no | Block until site reaches active status |
| `--poll-interval` | no | How often to poll for status (default 60s, min 1s) |
| `--timeout` | no | Maximum time to wait (default 5m, max 30m) |

#### vector site update

```
vector site update <site-id> [--customer-id <id>] [--tags <t1,t2>]
```

Updates site metadata. Only flags that are passed are included (PATCH semantics).
Empty `--tags ""` clears tags.

#### vector site delete

```
vector site delete <site-id> [--force]
```

Deletes a site (irreversible). Requires confirmation unless `--force`.

#### vector site clone

```
vector site clone <site-id> [--customer-id <id>] [--php-version <ver>] [--tags <t1,t2>]
```

Clones an existing site with files and database. Returns new DB credentials.

#### vector site suspend / unsuspend

```
vector site suspend <site-id>
vector site unsuspend <site-id>
```

Suspend or resume a site's development container.

#### vector site purge-cache

```
vector site purge-cache <site-id> [--cache-tag <tag>] [--url <url>]
```

Purges CDN cache. Optionally filter by cache tag or specific URL.

#### vector site logs

```
vector site logs <site-id> --environment <name> \
  [--start-time <time>] [--end-time <time>] \
  [--limit N] [--level <level>] \
  [--deployment-id <id>] [--cursor <cursor>]
```

Retrieves site logs for an environment. `--environment` is required. Time values
accept RFC3339 or relative format (e.g., `now-1h`).

| Flag | Type | Description |
|------|------|-------------|
| `--environment` | string | Environment name, e.g. prod, staging (required) |
| `--start-time` | string | Start time (RFC3339 or relative) |
| `--end-time` | string | End time (RFC3339 or relative) |
| `--limit` | int | Number of entries (1-1000) |
| `--level` | string | Filter: error, warning, info |
| `--deployment-id` | string | Filter by deployment |
| `--cursor` | string | Pagination cursor |

#### vector site reset-sftp-password / reset-db-password

```
vector site reset-sftp-password <site-id>
vector site reset-db-password <site-id>
```

Generates new credentials (shown once).

#### vector site wp-reconfig

```
vector site wp-reconfig <site-id>
```

Regenerates `wp-config.php`.

#### vector site ssh-key list / add / remove

```
vector site ssh-key list <site-id> [--page N] [--per-page N]
vector site ssh-key add <site-id> --name <name> --public-key <key>
vector site ssh-key remove <site-id> <key-id>
```

Manage SSH keys for a specific site.

---

### Environments

#### vector env list

```
vector env list <site-id> [--page N] [--per-page N] [--json]
```

Lists environments. Columns: ID, NAME, PRODUCTION, STATUS, PHP, PLATFORM DOMAIN, CUSTOM DOMAIN.

#### vector env show

```
vector env show <env-id> [--json]
```

Displays environment details including certificate status.

#### vector env create

```
vector env create <site-id> --name <name> --php-version <ver> \
  [--custom-domain <domain>] [--production] [--tags <t1,t2>]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--name` | yes | Environment name (slug format) |
| `--php-version` | yes | PHP version |
| `--custom-domain` | no | Custom domain |
| `--production` | no | Mark as production (default false) |
| `--tags` | no | Comma-separated tags |

#### vector env update

```
vector env update <env-id> [--custom-domain <domain>] [--clear-custom-domain] [--tags <t1,t2>]
```

`--custom-domain` and `--clear-custom-domain` are mutually exclusive.
Returns 202 Accepted if a domain change triggers an async infrastructure update.

#### vector env delete

```
vector env delete <env-id> [--force]
```

Deletes environment (irreversible). Requires confirmation unless `--force`.

#### vector env secret list / show / create / update / delete

```
vector env secret list <env-id> [--page N] [--per-page N]
vector env secret show <secret-id>
vector env secret create <env-id> --key <key> --value <val> [--is-secret]
vector env secret update <secret-id> [--key <key>] [--value <val>] [--is-secret]
vector env secret delete <secret-id> [--force]
```

Manage environment-level secrets and variables. `--is-secret` defaults to true.

#### vector env db promote / promote-status

```
vector env db promote <env-id> [--drop-tables] [--disable-foreign-keys]
vector env db promote-status <env-id> <promote-id>
```

Promotes the development database to the environment. Both flags default to true.

---

### Deployments

#### vector deploy list

```
vector deploy list <env-id> [--page N] [--per-page N]
```

Columns: ID, STATUS, ACTOR, CREATED.

#### vector deploy show

```
vector deploy show <deploy-id> [--json]
```

Shows deployment details including stdout/stderr.

#### vector deploy trigger

```
vector deploy trigger <env-id> [--include-uploads] [--include-database] \
  [--wait] [--poll-interval <duration>] [--timeout <duration>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--include-uploads` | false | Include uploads in deployment |
| `--include-database` | true | Include database in deployment |
| `--wait` | false | Block until deployment reaches a terminal status |
| `--poll-interval` | 60s | How often to poll for status (min 1s) |
| `--timeout` | 5m | Maximum time to wait (max 30m) |

#### vector deploy rollback

```
vector deploy rollback <env-id> [--target <deploy-id>] \
  [--wait] [--poll-interval <duration>] [--timeout <duration>]
```

Rolls back to last successful deployment, or to a specific `--target`.

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | | Specific deployment ID to roll back to |
| `--wait` | false | Block until rollback deployment reaches a terminal status |
| `--poll-interval` | 60s | How often to poll for status (min 1s) |
| `--timeout` | 5m | Maximum time to wait (max 30m) |

---

### SSL Certificates

#### vector ssl status

```
vector ssl status <env-id> [--json]
```

Shows SSL provisioning status, step, failure reason, and domains.

#### vector ssl nudge

```
vector ssl nudge <env-id> [--retry]
```

Manually nudges SSL provisioning for stuck or failed states.

---

### Backups

#### vector backup list

```
vector backup list [--site-id <id>] [--environment-id <id>] [--type <type>] \
  [--page N] [--per-page N]
```

Columns: ID, MODEL, TYPE, SCOPE, STATUS, DESCRIPTION, CREATED.

#### vector backup show

```
vector backup show <id> [--json]
```

#### vector backup create

```
vector backup create --site-id <id> [--environment-id <id>] \
  [--scope <full|database|files>] [--description <desc>]
```

`--site-id` or `--environment-id` required. `--scope` defaults to `full`.

#### vector backup download create / status

```
vector backup download create <backup-id>
vector backup download status <backup-id> <download-id>
```

Creates a download request, then poll status until a URL is returned.

---

### Restores

#### vector restore list

```
vector restore list [--site-id <id>] [--environment-id <id>] \
  [--type <site|environment>] [--backup-id <id>] [--page N] [--per-page N]
```

#### vector restore show

```
vector restore show <id> [--json]
```

#### vector restore create

```
vector restore create <backup-id> [--drop-tables] [--disable-foreign-keys] \
  [--search-replace-from <url>] [--search-replace-to <url>] \
  [--wait] [--poll-interval <duration>] [--timeout <duration>]
```

Initiates a restore from backup. `--drop-tables` and `--disable-foreign-keys`
default to false.

| Flag | Default | Description |
|------|---------|-------------|
| `--wait` | false | Block until restore reaches a terminal status |
| `--poll-interval` | 60s | How often to poll for status (min 1s) |
| `--timeout` | 5m | Maximum time to wait (max 30m) |

---

### WAF (Web Application Firewall)

#### Blocked IPs

```
vector waf blocked-ip list <site-id>
vector waf blocked-ip add <site-id> <ip>
vector waf blocked-ip remove <site-id> <ip>
```

#### Blocked Referrers

```
vector waf blocked-referrer list <site-id>
vector waf blocked-referrer add <site-id> <hostname>
vector waf blocked-referrer remove <site-id> <hostname>
```

#### Allowed Referrers

```
vector waf allowed-referrer list <site-id>
vector waf allowed-referrer add <site-id> <hostname>
vector waf allowed-referrer remove <site-id> <hostname>
```

#### Rate Limits

```
vector waf rate-limit list <site-id>
vector waf rate-limit show <site-id> <rule-id>
vector waf rate-limit create <site-id> --name <name> --request-count <N> \
  --timeframe <sec> --block-time <sec> [--description <desc>] \
  [--value <pattern>] [--operator <op>] [--variables <v1,v2>] \
  [--transformations <t1,t2>]
vector waf rate-limit update <site-id> <rule-id> [--name <name>] \
  [--request-count <N>] [--timeframe <sec>] [--block-time <sec>] \
  [--description <desc>] [--value <pattern>] [--operator <op>] \
  [--variables <v1,v2>] [--transformations <t1,t2>]
vector waf rate-limit delete <site-id> <rule-id>
```

---

### Database Operations

#### vector db export create / status

```
vector db export create <site-id> [--format sql]
vector db export status <site-id> <export-id>
```

Creates a SQL dump, then poll status for the download URL.

#### vector db import-session create / run / status

```
vector db import-session create <site-id> --content-length <bytes> \
  [--filename <name>] [--drop-tables] [--disable-foreign-keys] \
  [--search-replace-from <from>] [--search-replace-to <to>]
vector db import-session run <site-id> <import-id> [--parts '<json>']
vector db import-session status <site-id> <import-id>
```

Three-step import: create session (get upload URL), upload file, run import.
For files >= 5GB, the create response includes multipart upload details
(`is_multipart`, `upload_parts`). Use `--json` to get presigned URLs for each
part. After uploading all parts, pass the completed ETags to the run command
with `--parts`.

#### vector archive import

```
vector archive import <site-id> <file> [--drop-tables] [--disable-foreign-keys] \
  [--search-replace-from <from>] [--search-replace-to <to>]
```

One-command archive import: creates session, uploads file, and runs import.
Files >= 5GB are automatically uploaded using S3 multipart upload.

---

### Events

#### vector event list

```
vector event list [--from <ISO-8601>] [--to <ISO-8601>] [--event <type>] \
  [--page N] [--per-page N]
```

Lists account events. Columns: ID, EVENT, ACTOR, RESOURCE, CREATED.

---

### Webhooks

#### vector webhook list / show / create / update / delete

```
vector webhook list [--page N] [--per-page N]
vector webhook show <id>
vector webhook create --url <url> --events <e1,e2> [--type <http|slack>]
vector webhook update <id> [--url <url>] [--events <e1,e2>] [--enabled]
vector webhook delete <id>
```

`--type` defaults to `http`. Create returns a secret (shown once).

---

### Account

#### vector account show

```
vector account show [--json]
```

Displays owner, account name, company, resource counts.

#### vector account ssh-key list / show / create / delete

```
vector account ssh-key list [--page N] [--per-page N]
vector account ssh-key show <key-id>
vector account ssh-key create --name <name> --public-key <key>
vector account ssh-key delete <key-id>
```

#### vector account api-key list / create / delete

```
vector account api-key list [--page N] [--per-page N]
vector account api-key create --name <name> [--abilities <a1,a2>] [--expires-at <ISO-8601>]
vector account api-key delete <key-id>
```

Create returns a token (shown once).

#### vector account secret list / show / create / update / delete

```
vector account secret list [--page N] [--per-page N]
vector account secret show <id>
vector account secret create --key <key> --value <val> [--no-secret]
vector account secret update <id> [--value <val>] [--no-secret]
vector account secret delete <id>
```

`--no-secret` stores as a plain (non-secret) variable.

---

### Utilities

#### vector api

```
vector api <endpoint> [--method <verb>] [--raw-field <k=v>] [--field <k=v>] [--input <file|->] [--json] [--jq <expr>]
```

Sends an authenticated request to any Vector Pro API endpoint and prints the
raw response. Use it to reach endpoints that have no dedicated subcommand. An
`<endpoint>` beginning with `/` is sent verbatim against the base URL; any other
value has `/api/v1/vector/` prepended, so `sites` resolves to
`/api/v1/vector/sites`. JSON responses are pretty-printed and honor `--jq`
(which operates on the full envelope, including `data`/`meta`); non-JSON bodies
are written verbatim.

Flags:

- `--method`, `-X` — HTTP method. Defaults to GET, or POST when any field or
  `--input` is given.
- `--raw-field`, `-f` — add a **string** parameter in `key=value` form
  (repeatable).
- `--field`, `-F` — add a **typed** parameter in `key=value` form (repeatable):
  `true`/`false`/`null` and numeric literals become JSON types; `@file` loads
  the value from a file and `@-` reads it from stdin. Reusing a plain key is an
  error (exit code 3); use the `key[]=value` suffix to build an array. For
  POST/PUT/PATCH/DELETE, fields are sent as a JSON body; for GET they become
  query parameters.
- `--input` — send a raw request body from a file, or from stdin when set to
  `-`. Mutually exclusive with `-f`/`-F`.
- `--include`, `-i` — print the response status line and headers before the
  body.
- `--verbose` — echo the resolved request (method, URL, body) to stderr before
  sending; stdout is unchanged.

```bash
# GET an endpoint with no dedicated subcommand
vector api php-versions

# Equivalent absolute path
vector api /api/v1/vector/php-versions

# Filter the full envelope with built-in jq
vector api sites --jq '.data[].id'

# Create a resource with typed fields (auto-POST)
vector api sites -f your_customer_id=cust_123 -F dev_php_version=8.3

# Send a raw JSON body from a file or stdin
vector api sites --method POST --input body.json
echo '{"your_customer_id":"cust_123"}' | vector api sites -X POST --input -

# Inspect the response status line and headers
vector api sites -i

# Echo the resolved request to stderr before sending
vector api sites --verbose
```

#### vector php-versions

```
vector php-versions [--json]
```

Lists available PHP versions.

#### vector mcp setup

```
vector mcp setup [--target <desktop|code>] [--global] [--force]
```

Configures Vector MCP server for Claude Desktop or Claude Code.

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | desktop | Target: `desktop` or `code` |
| `--global` | false | For Code: write to `~/.claude.json` instead of `.mcp.json` |
| `--force` | false | Overwrite existing configuration |

---

## Common Workflows

### Deploy a Site

```bash
# Single blocking call (recommended for agents)
vector deploy trigger <env-id> --wait --json

# Or manually poll
vector deploy trigger <env-id> --json
vector deploy show <deploy-id> --json

# Rollback if needed (blocking)
vector deploy rollback <env-id> --wait --json
```

### Create a Site and Wait for Active

```bash
# Single blocking call — credentials are merged into the final JSON
vector site create --customer-id <id> --wait --json

# With custom timeout for large sites
vector site create --customer-id <id> --wait --timeout 15m --json
```

### Backup and Restore

```bash
# 1. Create backup
vector backup create --site-id <site-id> --scope full --json

# 2. Download backup
vector backup download create <backup-id> --json
vector backup download status <backup-id> <download-id> --json

# 3. Restore from backup (blocking)
vector restore create <backup-id> --wait --json

# Or manually poll
vector restore create <backup-id> --json
vector restore show <restore-id> --json
```

### WAF: Block an IP

```bash
vector waf blocked-ip add <site-id> <ip>
vector waf blocked-ip list <site-id> --json
```

### SSL Troubleshooting

```bash
# Check SSL status
vector ssl status <env-id> --json

# Nudge if stuck
vector ssl nudge <env-id> --retry
```

### Database Export/Import

```bash
# Export
vector db export create <site-id> --json
vector db export status <site-id> <export-id> --json

# Import (one-command)
vector archive import <site-id> dump.sql

# Import (multi-step)
vector db import-session create <site-id> --content-length 52428800 --filename dump.sql --json
# Upload file to the returned presigned URL (single or multipart)
vector db import-session run <site-id> <import-id>
# For multipart uploads, pass completed parts:
# vector db import-session run <site-id> <import-id> --parts '[{"part_number":1,"etag":"\"...\""},...]'
vector db import-session status <site-id> <import-id> --json
```

### Environment Management

```bash
# Create staging environment
vector env create <site-id> --name staging --php-version 8.3 --json

# Set a custom domain
vector env update <env-id> --custom-domain staging.example.com

# Add environment secret
vector env secret create <env-id> --key DB_PASSWORD --value secret123

# Promote dev database
vector env db promote <env-id>
```

---

## Decision Trees

### Which deploy command?

```
Need to deploy code?
├── Yes → vector deploy trigger <env-id>
│   ├── Include uploads? → --include-uploads
│   ├── Skip database? → --include-database=false
│   └── Wait for completion? → --wait [--timeout 10m]
└── Need to undo? → vector deploy rollback <env-id>
    ├── Specific version? → --target <deploy-id>
    └── Wait for completion? → --wait
```

### Which backup/restore path?

```
Need a backup?
├── Create → vector backup create --site-id <id>
│   ├── Full → --scope full (default)
│   ├── Database only → --scope database
│   └── Files only → --scope files
├── Download → vector backup download create <backup-id>
│   └── Poll → vector backup download status <backup-id> <download-id>
└── Restore → vector restore create <backup-id>
    ├── With search-replace → --search-replace-from/--search-replace-to
    └── Wait for completion? → --wait [--timeout 10m]
```

### Which WAF command?

```
WAF action needed?
├── Block IP → vector waf blocked-ip add <site-id> <ip>
├── Block referrer → vector waf blocked-referrer add <site-id> <hostname>
├── Allow referrer → vector waf allowed-referrer add <site-id> <hostname>
└── Rate limit → vector waf rate-limit create <site-id> --name ... --request-count ... --timeframe ... --block-time ...
```

### Which database operation?

```
Database operation?
├── Export → vector db export create <site-id>
│   └── Poll → vector db export status <site-id> <export-id>
├── Import (simple) → vector archive import <site-id> <file>
├── Import (multi-step) → vector db import-session create/run/status
└── Promote dev → vector env db promote <env-id>
```

---

## Error Handling

### Authentication Errors (exit code 2)

Token is missing, expired, or lacks permissions. Check with:
```bash
vector auth status --json
```

### Validation Errors (exit code 3)

Request payload failed server-side validation. The error message includes
field-level details in the format `field: message`.

### Not Found (exit code 4)

The referenced resource ID does not exist or is not accessible with the
current token.

### Server Errors (exit code 5)

Transient API error. Retry after a brief delay.

### General Errors (exit code 1)

Client-side error (network failure, invalid flags, etc.).
