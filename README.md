<p align="center">
    <img alt="BuiltFast Logo Light Mode" src="/assets/images/logo-light-mode.svg#gh-light-mode-only"/>
    <img alt="BuiltFast Logo Dark Mode" src="/assets/images/logo-dark-mode.svg#gh-dark-mode-only"/>
</p>

# Vector CLI

Official command-line interface for [Vector Pro](https://builtfast.dev/api) by [BuiltFast](https://builtfast.com).

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

## Installation

### Homebrew (macOS)

```bash
brew install built-fast/devtools/vector
```

### Pre-built binaries

Download from [Releases](https://github.com/built-fast/vector-cli/releases):

| Platform | Architecture | File |
|----------|--------------|------|
| Linux | x86_64 | `vector_VERSION_linux_amd64.tar.gz` |
| Linux | ARM64 | `vector_VERSION_linux_arm64.tar.gz` |
| macOS | x86_64 (Intel) | `vector_VERSION_darwin_amd64.tar.gz` |
| macOS | ARM64 (Apple Silicon) | `vector_VERSION_darwin_arm64.tar.gz` |
| Windows | x86_64 | `vector_VERSION_windows_amd64.zip` |

```bash
# Example: Linux x86_64
curl -LO https://github.com/built-fast/vector-cli/releases/latest/download/vector_VERSION_linux_amd64.tar.gz
tar xzf vector_VERSION_linux_amd64.tar.gz
sudo mv vector /usr/local/bin/
```

### From source

Requires [Go](https://go.dev/) 1.26+.

```bash
go install github.com/built-fast/vector-cli/cmd/vector@latest
```

Or build from a local clone:

```bash
make build
# Binary is at ./bin/vector
```

### Shell Completions

```bash
# Bash (add to ~/.bashrc)
eval "$(vector completion bash)"

# Zsh (add to ~/.zshrc)
eval "$(vector completion zsh)"

# Fish
vector completion fish | source
# To load on startup:
vector completion fish > ~/.config/fish/completions/vector.fish

# PowerShell (add to $PROFILE)
vector completion powershell | Out-String | Invoke-Expression
```

Homebrew installs completions automatically.

## Usage

### Global Flags

```bash
vector --token YOUR_TOKEN <command>  # Use a specific API token for this invocation
vector --json <command>              # Force JSON output
vector --no-json <command>           # Force table output
vector --jq <expr> <command>         # Filter JSON output with a jq expression
vector --version                     # Print version
```

### Authentication

```bash
# Login with API token (interactive prompt)
vector auth login

# Login with token from environment or argument
vector auth login --token YOUR_TOKEN
VECTOR_API_KEY=YOUR_TOKEN vector auth login

# Check auth status
vector auth status

# Logout
vector auth logout
```

### Sites

```bash
# List and view sites
vector site list
vector site show <site_id>

# Create and manage sites
vector site create --customer-id <id> --dev-php-version 8.3 [--production-domain example.com] [--staging-domain staging.example.com] [--tags tag1,tag2] [--wp-admin-email admin@example.com] [--wp-admin-user myadmin] [--wp-site-title "My Blog"]
vector site update <site_id> [--customer-id <id>] [--tags tag1,tag2]
vector site delete <site_id>
vector site clone <site_id> [--customer-id <id>] [--dev-php-version 8.3]

# Site operations
vector site suspend <site_id>
vector site unsuspend <site_id>
vector site reset-sftp-password <site_id>
vector site reset-db-password <site_id>
vector site purge-cache <site_id> [--cache-tag <tag>] [--url <url>]
vector site wp-reconfig <site_id>
vector site logs <site_id> [--start-time <time>] [--end-time <time>] [--limit 100]
```

### Site SSH Keys

```bash
vector site ssh-key list <site_id>
vector site ssh-key add <site_id> --name "My Key" --public-key "ssh-rsa ..."
vector site ssh-key remove <site_id> <key_id>
```

### Environments

```bash
# List and view environments
vector env list <site_id>
vector env show <env_id>

# Create and manage environments
vector env create <site_id> --name staging --custom-domain example.com --php-version 8.3 [--is-production]
vector env update <env_id> [--custom-domain <domain>] [--clear-custom-domain] [--tags tag1,tag2]
vector env delete <env_id>

# Reset database password
vector env reset-db-password <env_id>

# Check domain change status
vector env domain-change-status <env_id> <domain_change_id>
```

### Environment Secrets

```bash
vector env secret list <env_id>
vector env secret show <secret_id>
vector env secret create <env_id> --key MY_SECRET --value "secret-value" [--no-secret]
vector env secret update <secret_id> [--key <key>] [--value <value>] [--no-secret]
vector env secret delete <secret_id>
```

### Environment Database

```bash
# Promote dev database to environment
vector env db promote <env_id> [--drop-tables] [--disable-foreign-keys]
vector env db promote-status <env_id> <promote_id>
```

### Deployments

```bash
vector deploy list <env_id>
vector deploy show <deploy_id>
vector deploy trigger <env_id> [--include-uploads] [--include-database]
vector deploy rollback <env_id> [--target-deployment-id <id>]
```

### SSL

```bash
vector ssl status <env_id>
vector ssl nudge <env_id> [--retry]
```

### Database

```bash
# Import session for large files
vector db import-session create <site_id> [--filename <name>] [--content-length <bytes>] [--drop-tables] [--disable-foreign-keys] [--search-replace-from <from>] [--search-replace-to <to>]
vector db import-session run <site_id> <import_id> [--parts '<json>']
vector db import-session status <site_id> <import_id>

# Export
vector db export create <site_id>
vector db export status <site_id> <export_id>
```

When `--content-length` exceeds 5GB, the API returns multipart upload details
instead of a single upload URL. Use `--json` to see all part URLs. After
uploading each part, pass the ETags to the run command:

```bash
vector db import-session run <site_id> <import_id> --parts '[{"part_number":1,"etag":"\"abc...\""},...]'
```

### Archives

```bash
vector archive import <site_id> <file.tar.gz> [--drop-tables] [--disable-foreign-keys] [--search-replace-from <from>] [--search-replace-to <to>] [--wait] [--poll-interval <seconds>]
```

Files larger than 5GB are automatically uploaded using S3 multipart upload.
The CLI handles splitting the file into parts, uploading each one, and
finalizing the upload — no additional flags are needed.

### Backups

```bash
vector backup list [--site-id <id>] [--environment-id <id>] [--type site|environment]
vector backup show <backup_id>
vector backup create [--site-id <id>] [--environment-id <id>] [--scope full|database|files] [--description <desc>]
vector backup download create <backup_id>
vector backup download status <backup_id> <download_id>
```

### Restores

```bash
vector restore list [--site-id <id>] [--environment-id <id>] [--type site|environment] [--backup-id <id>]
vector restore show <restore_id>
vector restore create <backup_id> [--scope full|database|files] [--drop-tables] [--disable-foreign-keys] [--search-replace-from <from>] [--search-replace-to <to>]
```

### WAF

```bash
# Rate limits
vector waf rate-limit list <site_id>
vector waf rate-limit show <site_id> <rule_id>
vector waf rate-limit create <site_id> --name "Limit" --request-count 100 --timeframe 60 --block-time 300
vector waf rate-limit update <site_id> <rule_id> [--name <name>] [--request-count <n>]
vector waf rate-limit delete <site_id> <rule_id>

# Blocked IPs
vector waf blocked-ip list <site_id>
vector waf blocked-ip add <site_id> <ip>
vector waf blocked-ip remove <site_id> <ip>

# Blocked referrers
vector waf blocked-referrer list <site_id>
vector waf blocked-referrer add <site_id> <hostname>
vector waf blocked-referrer remove <site_id> <hostname>

# Allowed referrers
vector waf allowed-referrer list <site_id>
vector waf allowed-referrer add <site_id> <hostname>
vector waf allowed-referrer remove <site_id> <hostname>
```

### Account

```bash
# Account summary
vector account show

# Account SSH keys
vector account ssh-key list
vector account ssh-key show <key_id>
vector account ssh-key create --name "My Key" --public-key "ssh-rsa ..."
vector account ssh-key delete <key_id>

# API keys
vector account api-key list
vector account api-key create --name "CI Token" [--abilities read,write] [--expires-at 2025-12-31]
vector account api-key delete <token_id>

# Global secrets
vector account secret list
vector account secret show <secret_id>
vector account secret create --key MY_SECRET --value "secret-value" [--no-secret]
vector account secret update <secret_id> [--key <key>] [--value <value>] [--no-secret]
vector account secret delete <secret_id>
```

### Events

```bash
vector event list [--from 2024-01-01] [--to 2024-12-31] [--event site.created]
```

### Webhooks

```bash
vector webhook list
vector webhook show <webhook_id>
vector webhook create --name "My Webhook" --url "https://example.com/hook" --events site.created,deployment.completed
vector webhook update <webhook_id> [--name <name>] [--url <url>] [--enabled true]
vector webhook delete <webhook_id>
```

### PHP Versions

```bash
vector php-versions
```

### MCP Integration

Configure [Claude Desktop](https://claude.ai/download) to use Vector CLI as an MCP server:

```bash
vector mcp setup
```

## Output Format

- **Interactive (TTY)**: Human-readable table format
- **Piped/scripted**: JSON format

Override with flags:

```bash
vector site list --json          # Force JSON
vector site list --no-json       # Force table
vector site list | jq '.data'    # Auto JSON when piped
```

### JQ Filtering

The `--jq` flag filters JSON output using a built-in jq processor (no external `jq` binary required). It automatically forces JSON output.

```bash
# Extract specific fields
vector site list --jq '.[].id'
vector site show 456 --jq '.dev_domain'

# Filter with select
vector env list --site-id 123 --jq '[.[] | select(.status == "active")]'

# Count items
vector webhook list --jq 'length'
```

#### Format Strings

The `--jq` flag supports jq format strings for converting values:

```bash
# CSV output
vector site list --jq '[.[] | [.id, .name]] | .[] | @csv'

# TSV output
vector site list --jq '[.[] | [.id, .name]] | .[] | @tsv'

# URL-encode a value
vector site show 456 --jq '.name | @uri'

# Base64-encode a value
vector site show 456 --jq '.name | @base64'
```

Supported format strings: `@csv`, `@tsv`, `@html`, `@uri`, `@base64`.

## Configuration

Configuration is stored in `~/.config/vector/` (XDG-compliant):

- `config.json` - Optional settings
- API token is stored in the system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)

### Environment Variables

| Variable | Description |
|----------|-------------|
| `VECTOR_API_KEY` | API token (overrides keyring) |
| `VECTOR_API_URL` | API base URL (default: `https://api.builtfast.com`) |
| `VECTOR_CONFIG_DIR` | Config directory (default: `~/.config/vector`) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication error (401, 403) |
| 3 | Validation error (422) |
| 4 | Not found (404) |
| 5 | Network/server error (5xx) |

## Development

Requires [Go](https://go.dev/) 1.26+ and [golangci-lint](https://golangci-lint.run/).

```bash
make build      # Build binary to ./bin/vector
make test       # Run unit tests
make lint       # Run golangci-lint
make test-e2e   # Run end-to-end tests
make check      # Run lint + test + test-e2e
make clean      # Remove build artifacts
```

## License

MIT - see [LICENSE](LICENSE) for details.
