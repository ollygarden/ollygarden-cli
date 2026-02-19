package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// KVPair represents a key-value pair for single-resource output.
type KVPair struct {
	Key   string
	Value string
}

// Formatter handles all CLI output.
type Formatter struct {
	json      bool
	quiet     bool
	writer    io.Writer // stdout
	errWriter io.Writer // stderr
}

// New creates a Formatter using the given writers for data and errors.
func New(out, errOut io.Writer, jsonMode, quiet bool) *Formatter {
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	return &Formatter{
		json:      jsonMode,
		quiet:     quiet,
		writer:    out,
		errWriter: errOut,
	}
}

// PrintJSON writes raw JSON to stdout.
func (f *Formatter) PrintJSON(data json.RawMessage) {
	fmt.Fprintln(f.writer, string(data))
}

// PrintTable writes an aligned table to stdout.
func (f *Formatter) PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// PrintKeyValue writes key-value pairs to stdout.
func (f *Formatter) PrintKeyValue(pairs []KVPair) {
	// Find max key length for alignment
	maxLen := 0
	for _, p := range pairs {
		if len(p.Key) > maxLen {
			maxLen = len(p.Key)
		}
	}
	for _, p := range pairs {
		fmt.Fprintf(f.writer, "%-*s  %s\n", maxLen+1, p.Key+":", p.Value)
	}
}

// PrintError writes an error to stderr. In JSON mode, writes the full error envelope.
func (f *Formatter) PrintError(msg string, jsonData json.RawMessage) {
	if f.json && jsonData != nil {
		fmt.Fprintln(f.errWriter, string(jsonData))
		return
	}
	fmt.Fprintln(f.errWriter, msg)
}

// PrintPaginationHint writes a pagination hint to stderr when there are more results.
func (f *Formatter) PrintPaginationHint(total, offset, limit int) {
	if f.quiet {
		return
	}
	remaining := total - offset - limit
	if remaining <= 0 {
		return
	}
	nextOffset := offset + limit
	fmt.Fprintf(f.errWriter, "# %d more results. Use --offset %d to see next page.\n", remaining, nextOffset)
}

// IsQuiet returns whether quiet mode is enabled.
func (f *Formatter) IsQuiet() bool {
	return f.quiet
}

// IsJSON returns whether JSON mode is enabled.
func (f *Formatter) IsJSON() bool {
	return f.json
}
