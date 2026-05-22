package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// KeyValue represents a key-value pair for display in show commands.
type KeyValue struct {
	Key   string
	Value string
}

// PrintJSON writes v as pretty-printed (indented) JSON to w.
func PrintJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintTable writes a formatted table with headers and rows to w.
func PrintTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, h)
	}
	_, _ = fmt.Fprintln(tw)

	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			_, _ = fmt.Fprint(tw, col)
		}
		_, _ = fmt.Fprintln(tw)
	}
	_ = tw.Flush()
}

// PrintKeyValue writes key-value pairs with a right-aligned key column to w.
func PrintKeyValue(w io.Writer, pairs []KeyValue) {
	maxLen := 0
	for _, p := range pairs {
		if len(p.Key) > maxLen {
			maxLen = len(p.Key)
		}
	}
	for _, p := range pairs {
		_, _ = fmt.Fprintf(w, "%*s: %s\n", maxLen, p.Key, p.Value)
	}
}

// PrintPagination writes "Page X of Y (Z total)" to w.
func PrintPagination(w io.Writer, page, lastPage, total int) {
	_, _ = fmt.Fprintf(w, "Page %d of %d (%d total)\n", page, lastPage, total)
}

// PrintError writes "Error: <msg>" to w.
func PrintError(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "Error: %s\n", msg)
}

// PrintMessage writes a plain message line to w.
func PrintMessage(w io.Writer, msg string) {
	_, _ = fmt.Fprintln(w, msg)
}
