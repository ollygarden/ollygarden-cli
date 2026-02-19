package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestFormatter(jsonMode, quiet bool) (*Formatter, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	f := &Formatter{
		json:      jsonMode,
		quiet:     quiet,
		writer:    stdout,
		errWriter: stderr,
	}
	return f, stdout, stderr
}

func TestPrintJSON(t *testing.T) {
	f, stdout, _ := newTestFormatter(true, false)
	data := json.RawMessage(`{"data":{"id":"123"}}`)
	f.PrintJSON(data)
	assert.Equal(t, "{\"data\":{\"id\":\"123\"}}\n", stdout.String())
}

func TestPrintTable(t *testing.T) {
	f, stdout, _ := newTestFormatter(false, false)
	headers := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "svc-a", "active"},
		{"2", "svc-b", "inactive"},
	}
	f.PrintTable(headers, rows)
	out := stdout.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "svc-a")
	assert.Contains(t, out, "svc-b")
}

func TestPrintKeyValue(t *testing.T) {
	f, stdout, _ := newTestFormatter(false, false)
	f.PrintKeyValue([]KVPair{
		{Key: "ID", Value: "123"},
		{Key: "Name", Value: "test-svc"},
	})
	out := stdout.String()
	assert.Contains(t, out, "ID:")
	assert.Contains(t, out, "123")
	assert.Contains(t, out, "Name:")
	assert.Contains(t, out, "test-svc")
}

func TestPrintErrorHuman(t *testing.T) {
	f, _, stderr := newTestFormatter(false, false)
	f.PrintError("Error: not found", nil)
	assert.Equal(t, "Error: not found\n", stderr.String())
}

func TestPrintErrorJSON(t *testing.T) {
	f, _, stderr := newTestFormatter(true, false)
	errJSON := json.RawMessage(`{"error":{"code":"NOT_FOUND","message":"not found"}}`)
	f.PrintError("Error: not found", errJSON)
	assert.Contains(t, stderr.String(), "NOT_FOUND")
}

func TestPaginationHint(t *testing.T) {
	f, _, stderr := newTestFormatter(false, false)
	f.PrintPaginationHint(100, 0, 50)
	assert.Equal(t, "# 50 more results. Use --offset 50 to see next page.\n", stderr.String())
}

func TestPaginationHintQuiet(t *testing.T) {
	f, _, stderr := newTestFormatter(false, true)
	f.PrintPaginationHint(100, 0, 50)
	assert.Empty(t, stderr.String())
}

func TestPaginationHintNoMore(t *testing.T) {
	f, _, stderr := newTestFormatter(false, false)
	f.PrintPaginationHint(50, 0, 50)
	assert.Empty(t, stderr.String())
}

func TestIsQuiet(t *testing.T) {
	f, _, _ := newTestFormatter(false, true)
	assert.True(t, f.IsQuiet())
}

func TestIsJSON(t *testing.T) {
	f, _, _ := newTestFormatter(true, false)
	assert.True(t, f.IsJSON())
}
