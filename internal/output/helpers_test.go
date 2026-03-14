package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"name": "test", "count": 42}

	err := PrintJSON(&buf, data)
	require.NoError(t, err)

	expected := "{\n  \"count\": 42,\n  \"name\": \"test\"\n}\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintJSON_Slice(t *testing.T) {
	var buf bytes.Buffer
	data := []string{"a", "b"}

	err := PrintJSON(&buf, data)
	require.NoError(t, err)

	expected := "[\n  \"a\",\n  \"b\"\n]\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintJSON_InvalidValue(t *testing.T) {
	var buf bytes.Buffer
	// Channels cannot be marshaled to JSON.
	err := PrintJSON(&buf, make(chan int))
	assert.Error(t, err)
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"ID", "Name", "Status"}
	rows := [][]string{
		{"1", "Alpha", "active"},
		{"2", "Beta", "inactive"},
	}

	PrintTable(&buf, headers, rows)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "Alpha")
	assert.Contains(t, output, "Beta")
	assert.Contains(t, output, "active")
	assert.Contains(t, output, "inactive")
}

func TestPrintTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"ID", "Name"}

	PrintTable(&buf, headers, nil)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "Name")
}

func TestPrintKeyValue(t *testing.T) {
	var buf bytes.Buffer
	pairs := []KeyValue{
		{Key: "Name", Value: "Test Project"},
		{Key: "API URL", Value: "https://api.example.com"},
		{Key: "Status", Value: "active"},
	}

	PrintKeyValue(&buf, pairs)

	output := buf.String()
	// "API URL" is the longest key (7 chars), so shorter keys should be right-aligned.
	assert.Contains(t, output, "   Name: Test Project\n")
	assert.Contains(t, output, "API URL: https://api.example.com\n")
	assert.Contains(t, output, " Status: active\n")
}

func TestPrintKeyValue_SinglePair(t *testing.T) {
	var buf bytes.Buffer
	pairs := []KeyValue{
		{Key: "Key", Value: "Value"},
	}

	PrintKeyValue(&buf, pairs)
	assert.Equal(t, "Key: Value\n", buf.String())
}

func TestPrintPagination(t *testing.T) {
	var buf bytes.Buffer
	PrintPagination(&buf, 2, 5, 48)
	assert.Equal(t, "Page 2 of 5 (48 total)\n", buf.String())
}

func TestPrintPagination_SinglePage(t *testing.T) {
	var buf bytes.Buffer
	PrintPagination(&buf, 1, 1, 3)
	assert.Equal(t, "Page 1 of 1 (3 total)\n", buf.String())
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	PrintError(&buf, "something went wrong")
	assert.Equal(t, "Error: something went wrong\n", buf.String())
}

func TestPrintMessage(t *testing.T) {
	var buf bytes.Buffer
	PrintMessage(&buf, "Operation completed successfully.")
	assert.Equal(t, "Operation completed successfully.\n", buf.String())
}

func TestPrintMessage_EmptyString(t *testing.T) {
	var buf bytes.Buffer
	PrintMessage(&buf, "")
	assert.Equal(t, "\n", buf.String())
}
