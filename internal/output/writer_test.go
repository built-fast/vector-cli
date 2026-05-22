package output

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/itchyny/gojq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compileJQ is a test helper that parses and compiles a jq expression.
func compileJQ(t *testing.T, expr string) *gojq.Code {
	t.Helper()
	query, err := gojq.Parse(expr)
	require.NoError(t, err)
	code, err := gojq.Compile(query)
	require.NoError(t, err)
	return code
}

func TestWriter_JSON_WithoutJQ(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, JSON)

	err := w.JSON(map[string]any{"name": "test", "count": 42})
	require.NoError(t, err)

	expected := "{\n  \"count\": 42,\n  \"name\": \"test\"\n}\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriter_JSON_JQ_FieldAccess(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".name")
	w := NewWriter(&buf, JSON, WithJQ(".name", code))

	err := w.JSON(map[string]any{"name": "alice", "age": 30})
	require.NoError(t, err)

	assert.Equal(t, "alice\n", buf.String())
}

func TestWriter_JSON_JQ_ArrayFilter(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".[].id")
	w := NewWriter(&buf, JSON, WithJQ(".[].id", code))

	err := w.JSON([]any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	})
	require.NoError(t, err)

	assert.Equal(t, "1\n2\n", buf.String())
}

func TestWriter_JSON_JQ_Select(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, `[.[] | select(.active == true)]`)
	w := NewWriter(&buf, JSON, WithJQ(`[.[] | select(.active == true)]`, code))

	err := w.JSON([]any{
		map[string]any{"name": "a", "active": true},
		map[string]any{"name": "b", "active": false},
		map[string]any{"name": "c", "active": true},
	})
	require.NoError(t, err)

	expected := "[\n  {\n    \"active\": true,\n    \"name\": \"a\"\n  },\n  {\n    \"active\": true,\n    \"name\": \"c\"\n  }\n]\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriter_JSON_JQ_Length(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, "length")
	w := NewWriter(&buf, JSON, WithJQ("length", code))

	err := w.JSON([]any{1, 2, 3})
	require.NoError(t, err)

	assert.Equal(t, "3\n", buf.String())
}

func TestWriter_JSON_JQ_Pipe(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".[].name")
	w := NewWriter(&buf, JSON, WithJQ(".[].name", code))

	err := w.JSON([]any{
		map[string]any{"name": "alice"},
		map[string]any{"name": "bob"},
		map[string]any{"name": "charlie"},
	})
	require.NoError(t, err)

	assert.Equal(t, "alice\nbob\ncharlie\n", buf.String())
}

func TestWriter_JSON_JQ_Identity(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".")
	w := NewWriter(&buf, JSON, WithJQ(".", code))

	err := w.JSON(map[string]any{"x": 1})
	require.NoError(t, err)

	expected := "{\n  \"x\": 1\n}\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriter_JSON_JQ_FormatCSV(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, `[.[] | [.id, .name]] | .[] | @csv`)
	w := NewWriter(&buf, JSON, WithJQ(`[.[] | [.id, .name]] | .[] | @csv`, code))

	err := w.JSON([]any{
		map[string]any{"id": 1, "name": "alice"},
		map[string]any{"id": 2, "name": "bob"},
	})
	require.NoError(t, err)

	assert.Equal(t, "1,\"alice\"\n2,\"bob\"\n", buf.String())
}

func TestWriter_JSON_JQ_FormatBase64(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".name | @base64")
	w := NewWriter(&buf, JSON, WithJQ(".name | @base64", code))

	err := w.JSON(map[string]any{"name": "hello"})
	require.NoError(t, err)

	expected := base64.StdEncoding.EncodeToString([]byte("hello"))
	assert.Equal(t, expected+"\n", buf.String())
}

func TestWriter_JSON_JQ_PrettyPrintObject(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".info")
	w := NewWriter(&buf, JSON, WithJQ(".info", code))

	err := w.JSON(map[string]any{
		"info": map[string]any{"a": 1, "b": 2},
	})
	require.NoError(t, err)

	expected := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriter_JSON_JQ_PrettyPrintArray(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".items")
	w := NewWriter(&buf, JSON, WithJQ(".items", code))

	err := w.JSON(map[string]any{
		"items": []any{1, 2, 3},
	})
	require.NoError(t, err)

	expected := "[\n  1,\n  2,\n  3\n]\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriter_JSON_InvalidInput(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, JSON)

	err := w.JSON(make(chan int))
	assert.Error(t, err)
}

func TestWriter_JSON_JQ_RuntimeError(t *testing.T) {
	var buf bytes.Buffer
	// .foo on a number will produce a runtime error
	code := compileJQ(t, ".foo")
	w := NewWriter(&buf, JSON, WithJQ(".foo", code))

	err := w.JSON("not-an-object")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jq:")
}

func TestWriter_JSON_JQ_NilResult(t *testing.T) {
	var buf bytes.Buffer
	code := compileJQ(t, ".missing")
	w := NewWriter(&buf, JSON, WithJQ(".missing", code))

	err := w.JSON(map[string]any{"name": "test"})
	require.NoError(t, err)

	assert.Equal(t, "null\n", buf.String())
}

func TestWriter_HasJQ_True(t *testing.T) {
	code := compileJQ(t, ".")
	w := NewWriter(&bytes.Buffer{}, JSON, WithJQ(".", code))
	assert.True(t, w.HasJQ())
}

func TestWriter_HasJQ_False(t *testing.T) {
	w := NewWriter(&bytes.Buffer{}, JSON)
	assert.False(t, w.HasJQ())
}

func TestWriter_Format(t *testing.T) {
	w := NewWriter(&bytes.Buffer{}, Table)
	assert.Equal(t, Table, w.Format())

	w2 := NewWriter(&bytes.Buffer{}, JSON)
	assert.Equal(t, JSON, w2.Format())
}

func TestWriter_Table(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, Table)

	headers := []string{"ID", "Name"}
	rows := [][]string{{"1", "Alice"}, {"2", "Bob"}}
	w.Table(headers, rows)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "Bob")
}

func TestWriter_KeyValue(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, Table)

	pairs := []KeyValue{
		{Key: "Name", Value: "Test"},
		{Key: "Status", Value: "active"},
	}
	w.KeyValue(pairs)

	output := buf.String()
	assert.Contains(t, output, "Name: Test")
	assert.Contains(t, output, "Status: active")
}

func TestWriter_Message(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, Table)

	w.Message("hello world")
	assert.Equal(t, "hello world\n", buf.String())
}

func TestWriter_Error(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, Table)

	w.Error("something broke")
	assert.Equal(t, "Error: something broke\n", buf.String())
}

func TestWriter_Pagination(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, Table)

	w.Pagination(2, 5, 48)
	assert.Equal(t, "Page 2 of 5 (48 total)\n", buf.String())
}

func TestWriter_Underlying(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, JSON)
	assert.Equal(t, &buf, w.Underlying())
}
