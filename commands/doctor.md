---
description: Check Vector CLI health — binary, authentication, and live API connectivity.
allowed-tools: Bash(vector doctor:*)
---

# /vector:doctor

Run the Vector CLI health check and report the results.

```bash
vector doctor --json
```

The JSON output has an `ok` boolean and a `checks` array. Each check has a
`name`, a `status`, a `detail`, and an optional `hint`. Interpret the status:

- **pass** — working correctly
- **warn** — non-critical issue
- **skip** — check not run (e.g. unauthenticated)
- **fail** — broken, needs attention

The command always exits 0; rely on the `status` fields, not the exit code.

Common fixes (follow the `hint` field first):

- `auth` fail → run `vector auth login`, or set `VECTOR_API_KEY`
- `api` fail with "rejected" → token is invalid or expired; run `vector auth login`
- `api` fail with "network error" → check network/VPN connectivity

Report results concisely: list any failures and warnings with their hints. If
everything passes, say so in one line.
