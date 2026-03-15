package surface

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Generate walks a cobra command tree and produces a deterministic, sorted
// snapshot of all commands, flags, and positional arguments. Built-in commands
// (help, completion) and the --help flag are excluded.
func Generate(root *cobra.Command) string {
	var lines []string
	walk(root, &lines)
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

func walk(cmd *cobra.Command, lines *[]string) {
	name := cmd.Name()

	// Skip Cobra built-in commands.
	if name == "help" || name == "completion" {
		return
	}

	path := fullPath(cmd)

	// CMD line
	*lines = append(*lines, fmt.Sprintf("CMD %s", path))

	// ARG lines — extracted from the Use string.
	args := parseArgs(cmd.Use)
	for i, arg := range args {
		*lines = append(*lines, fmt.Sprintf("ARG %s %d %s", path, i, arg))
	}

	// FLAG lines.
	// For the root command, emit persistent flags (they apply globally).
	// For all commands, emit local non-persistent flags.
	if !cmd.HasParent() {
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			*lines = append(*lines, fmt.Sprintf("FLAG %s --%s type=%s", path, f.Name, f.Value.Type()))
		})
	}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		// Skip persistent flags (already emitted on root).
		if cmd.PersistentFlags().Lookup(f.Name) != nil {
			return
		}
		*lines = append(*lines, fmt.Sprintf("FLAG %s --%s type=%s", path, f.Name, f.Value.Type()))
	})

	for _, child := range cmd.Commands() {
		walk(child, lines)
	}
}

// fullPath returns the full command path (e.g. "vector site list").
func fullPath(cmd *cobra.Command) string {
	parts := []string{}
	for c := cmd; c != nil; c = c.Parent() {
		parts = append([]string{c.Name()}, parts...)
	}
	return strings.Join(parts, " ")
}

// parseArgs extracts positional argument names from a cobra Use string.
// e.g. "list <site-id>" -> ["site-id"], "add <site-id> <hostname>" -> ["site-id", "hostname"]
func parseArgs(use string) []string {
	var args []string
	for _, token := range strings.Fields(use) {
		if strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">") {
			args = append(args, token[1:len(token)-1])
		}
	}
	return args
}
