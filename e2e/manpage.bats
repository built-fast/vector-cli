#!/usr/bin/env bats
# manpage.bats - Verify man page documents all CLI commands

load test_helper

MANPAGE="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/man/man1/vector.1"

# Extract subcommand names from "Available Commands:" in --help output
extract_subcommands() {
  echo "$1" | sed -n '/^Available Commands:/,/^$/p' | \
    tail -n +2 | awk '{print $1}' | grep -v '^$'
}

# Check if help output indicates a group command (has Available Commands)
is_group() {
  echo "$1" | grep -q "^Available Commands:"
}

# Discover all leaf command paths recursively
discover_commands() {
  local prefix="$1"
  local help_output

  if [ -z "$prefix" ]; then
    help_output=$("$VECTOR_BINARY" --help 2>&1)
  else
    help_output=$("$VECTOR_BINARY" $prefix --help 2>&1)
  fi

  if is_group "$help_output"; then
    local subs
    subs=$(extract_subcommands "$help_output")
    for sub in $subs; do
      case "$sub" in help|completion) continue ;; esac
      if [ -z "$prefix" ]; then
        discover_commands "$sub"
      else
        discover_commands "$prefix $sub"
      fi
    done
  else
    [ -n "$prefix" ] && echo "$prefix"
  fi
}

@test "man page exists" {
  [ -f "$MANPAGE" ]
}

@test "all CLI commands are documented in the man page" {
  local commands
  commands=$(discover_commands "")

  local manpage_text
  manpage_text=$(cat "$MANPAGE")

  local missing=""
  local count=0

  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    count=$((count + 1))

    # Normalize hyphens: troff uses \- for literal hyphens
    local escaped
    escaped=$(echo "$cmd" | sed 's/-/\\\\-/g')

    if ! echo "$manpage_text" | grep -qE "^\.B ${escaped}( |$)" && \
       ! echo "$manpage_text" | grep -qE "^\.SS ${escaped} "; then
      missing="${missing}  ${cmd}\n"
    fi
  done <<< "$commands"

  if [ -n "$missing" ]; then
    echo "Commands missing from man page ($count total checked):"
    printf "$missing"
    return 1
  fi
}
