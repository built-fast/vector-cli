@STYLE.md

# Vector CLI Development Context

## Development Loop

Make changes, run `make check`, fix what it catches, repeat until green, then
push. `make check` runs lint + test + test-e2e. Treat it as your inner-loop
companion, not a final hurdle.

## Repository Structure

```
vector-cli/
├── cmd/vector/          # Main entrypoint (main.go)
├── internal/
│   ├── api/             # HTTP client and error handling
│   ├── appctx/          # Application context (App struct)
│   ├── cli/             # Root command wiring
│   ├── commands/        # Command implementations (one file per group)
│   ├── config/          # Configuration, keyring, paths
│   ├── output/          # Output formatting (Writer, Table, JSON, KeyValue)
│   └── version/         # Version info (injected via ldflags)
├── e2e/                 # BATS end-to-end tests (Prism mock server)
└── man/                 # Manpage generation
```

## Vector Pro API Reference

Base URL: `https://api.builtfast.com`

All resource paths are under `/api/v1/vector/`. Key resources:

- `/sites` — CRUD, suspend, unsuspend, clone, purge-cache, logs, wp-reconfig
- `/sites/{id}/environments` — environment management
- `/sites/{id}/environments/{id}/deployments` — deployments
- `/sites/{id}/backups` — backup management
- `/sites/{id}/backups/{id}/download` — backup download
- `/sites/{id}/restores` — restore management
- `/sites/{id}/waf/blocked-ips` — WAF blocked IPs
- `/sites/{id}/waf/blocked-referrers` — WAF blocked referrers
- `/sites/{id}/waf/allowed-referrers` — WAF allowed referrers
- `/sites/{id}/waf/rate-limits` — WAF rate limits
- `/sites/{id}/archives` — site archives
- `/sites/{id}/ssh-keys` — site SSH keys
- `/sites/{id}/db/export` — database export
- `/sites/{id}/db/import-sessions` — database import
- `/sites/{id}/events` — site events
- `/sites/{id}/environments/{id}/ssl` — SSL certificates
- `/sites/{id}/environments/{id}/secrets` — environment secrets
- `/sites/{id}/environments/{id}/db` — environment database info
- `/account` — account details
- `/account/ssh-keys` — account SSH keys
- `/account/api-keys` — API key management
- `/account/secrets` — account secrets
- `/webhooks` — webhook management
- `/php-versions` — available PHP versions
- `/auth/whoami` — authentication check
- `/mcp/config` — MCP server configuration

## Testing

`make check` is the local CI gate. Run it before pushing.

```bash
make check             # All checks: lint + test + test-e2e
make test              # Go unit tests only
make lint              # golangci-lint
make test-e2e          # BATS e2e tests (requires Prism)
make build             # Build binary to ./bin/vector
```

When iterating on a specific area, use targeted targets for faster feedback,
then finish with `make check` before pushing.

**E2E tests** use a [Prism](https://github.com/stoplightio/prism) mock server
that validates requests against `e2e/openapi.yaml`. The test helper (`e2e/test_helper.bash`)
starts Prism automatically.

**Requirements**: Go 1.26+, [golangci-lint](https://golangci-lint.run),
[bats-core](https://github.com/bats-core/bats-core), Node.js/npx (for Prism).
