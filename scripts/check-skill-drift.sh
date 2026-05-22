#!/usr/bin/env bash
set -eo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SURFACE="$REPO_ROOT/.surface"
SKILL="$REPO_ROOT/skills/vector/SKILL.md"
BASELINE="$REPO_ROOT/.surface-skill-drift"

# Strip YAML frontmatter (content between first pair of --- delimiters)
strip_frontmatter() {
    awk '
        /^---[[:space:]]*$/ { count++; next }
        count >= 2 { print }
    ' "$1"
}

# Resolve longest matching CMD in .surface for a "vector sub1 sub2..." pattern.
# Prints the resolved command path and returns 0, or returns 1 if no match.
resolve_cmd() {
    local words
    read -ra words <<< "$1"
    local i=${#words[@]}
    while (( i >= 1 )); do
        local try="${words[*]:0:i}"
        if grep -qx "CMD ${try}" "$SURFACE"; then
            echo "$try"
            return 0
        fi
        (( i-- ))
    done
    return 1
}

# Check if a flag exists on a command, its ancestors, or its descendants.
flag_exists() {
    local cmd="$1"
    local flag="$2"

    # Exact command
    grep -q "^FLAG ${cmd} ${flag} " "$SURFACE" && return 0

    # Descendants (subcommands of this command)
    grep -qE "^FLAG ${cmd} [a-z].* ${flag} " "$SURFACE" && return 0

    # Ancestors (inherited persistent flags, e.g. --json on root)
    local words
    read -ra words <<< "$cmd"
    local i=$(( ${#words[@]} - 1 ))
    while (( i >= 1 )); do
        local ancestor="${words[*]:0:i}"
        grep -q "^FLAG ${ancestor} ${flag} " "$SURFACE" && return 0
        (( i-- ))
    done

    return 1
}

# --- Main ---

content=$(strip_frontmatter "$SKILL")
drifts=()

# Phase 1: Extract "vector <subcommand>..." patterns and verify CMD exists
while IFS= read -r cmd_pattern; do
    [ -z "$cmd_pattern" ] && continue
    if ! resolve_cmd "$cmd_pattern" > /dev/null; then
        drifts+=("CMD: $cmd_pattern")
    fi
done < <(echo "$content" | { grep -oE 'vector( [a-z][a-z0-9-]*)+' || true; } | sort -u)

# Phase 2: For lines with "vector <cmd> ... --<flag>", verify flags exist
while IFS= read -r line; do
    [ -z "$line" ] && continue

    # Extract command path
    cmd_part=$(echo "$line" | grep -oE 'vector( [a-z][a-z0-9-]*)+' | head -1) || true
    [ -z "$cmd_part" ] && continue

    # Resolve to longest matching CMD
    resolved=$(resolve_cmd "$cmd_part") || continue

    # Check each flag on the line
    while IFS= read -r flag; do
        [ -z "$flag" ] && continue
        if ! flag_exists "$resolved" "$flag"; then
            drifts+=("FLAG: ${resolved} ${flag}")
        fi
    done < <(echo "$line" | { grep -oE -- '--[a-z][a-z0-9-]*' || true; } | sort -u)
done < <(echo "$content" | { grep -E 'vector [a-z].*--[a-z]' || true; })

# No drift found
if [ ${#drifts[@]} -eq 0 ]; then
    echo "No skill drift detected."
    exit 0
fi

# Deduplicate
drifts_deduped=()
while IFS= read -r d; do
    [ -z "$d" ] && continue
    drifts_deduped+=("$d")
done < <(printf '%s\n' "${drifts[@]}" | sort -u)
drifts=("${drifts_deduped[@]}")

# Filter out baselined drifts
new_drifts=()
for d in "${drifts[@]}"; do
    if [ -f "$BASELINE" ] && grep -qxF "$d" "$BASELINE"; then
        continue
    fi
    new_drifts+=("$d")
done

if [ ${#new_drifts[@]} -eq 0 ]; then
    echo "All drift is baselined. OK."
    exit 0
fi

echo "Skill drift detected (${#new_drifts[@]} issue(s)):"
for d in "${new_drifts[@]}"; do
    echo "  $d"
done
echo ""
echo "To baseline accepted mismatches, add them to .surface-skill-drift"
exit 1
