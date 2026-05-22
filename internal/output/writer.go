package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// WriterOption configures a Writer.
type WriterOption func(*Writer)

// WithJQ returns a WriterOption that enables jq filtering on JSON output.
func WithJQ(filter string, code *gojq.Code) WriterOption {
	return func(w *Writer) {
		w.jqFilter = filter
		w.jqCode = code
	}
}

// Writer wraps output formatting and optional jq filtering.
type Writer struct {
	w        io.Writer
	format   Format
	jqCode   *gojq.Code
	jqFilter string
}

// NewWriter creates a new Writer with the given io.Writer, format, and options.
func NewWriter(w io.Writer, format Format, opts ...WriterOption) *Writer {
	wr := &Writer{
		w:      w,
		format: format,
	}
	for _, opt := range opts {
		opt(wr)
	}
	return wr
}

// Format returns the configured output format.
func (wr *Writer) Format() Format {
	return wr.format
}

// HasJQ returns true when a jq filter is active.
func (wr *Writer) HasJQ() bool {
	return wr.jqCode != nil
}

// JSON writes v as pretty-printed JSON, applying a jq filter if set.
func (wr *Writer) JSON(v any) error {
	if wr.HasJQ() {
		return wr.writeJQ(v)
	}
	return PrintJSON(wr.w, v)
}

// Table delegates to PrintTable.
func (wr *Writer) Table(headers []string, rows [][]string) {
	PrintTable(wr.w, headers, rows)
}

// KeyValue delegates to PrintKeyValue.
func (wr *Writer) KeyValue(pairs []KeyValue) {
	PrintKeyValue(wr.w, pairs)
}

// Pagination delegates to PrintPagination.
func (wr *Writer) Pagination(page, lastPage, total int) {
	PrintPagination(wr.w, page, lastPage, total)
}

// Message delegates to PrintMessage.
func (wr *Writer) Message(msg string) {
	PrintMessage(wr.w, msg)
}

// Error delegates to PrintError.
func (wr *Writer) Error(msg string) {
	PrintError(wr.w, msg)
}

// Underlying returns the raw io.Writer.
func (wr *Writer) Underlying() io.Writer {
	return wr.w
}

// writeJQ marshals v to a generic value, runs the jq filter, and outputs results.
func (wr *Writer) writeJQ(v any) error {
	// Marshal then unmarshal to ensure we have a clean interface{} tree
	// that gojq can work with (no typed structs).
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var input any
	if err := json.Unmarshal(b, &input); err != nil {
		return err
	}

	iter := wr.jqCode.Run(input)
	for {
		result, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := result.(error); isErr {
			return fmt.Errorf("jq: %w", err)
		}

		switch val := result.(type) {
		case nil:
			_, _ = fmt.Fprintln(wr.w, "null")
		case string:
			_, _ = fmt.Fprintln(wr.w, val)
		default:
			out, err := json.MarshalIndent(val, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(wr.w, string(out))
		}
	}
	return nil
}
