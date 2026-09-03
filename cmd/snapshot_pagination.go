package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
)

type snapshotFlags struct {
	enabled  bool
	cursor   string
	all      bool
	maxPages int
}

func (flags snapshotFlags) validate(offset int, jsonOutput bool) error {
	if (flags.enabled || flags.cursor != "" || flags.all) && offset != 0 {
		return fmt.Errorf("--offset cannot be combined with --snapshot, --cursor, or --all; restart with --offset 0")
	}
	if flags.maxPages < 0 {
		return fmt.Errorf("--max-pages must be >= 0")
	}
	if flags.maxPages != 0 && !flags.all {
		return fmt.Errorf("--max-pages requires --all")
	}
	if flags.all && jsonOutput {
		return fmt.Errorf("--all cannot be combined with --json; use --snapshot and --cursor to preserve each API response envelope")
	}
	return nil
}

func (flags snapshotFlags) apply(query url.Values) {
	if flags.enabled || flags.all {
		query.Set("snapshot", "true")
	}
	if flags.cursor != "" {
		query.Set("cursor", flags.cursor)
	}
}

func printAPIError(f *output.Formatter, err error) {
	apiErr, ok := err.(*client.APIError)
	if !ok {
		return
	}
	var raw json.RawMessage
	if apiErr.ErrorResponse != nil {
		raw, _ = json.Marshal(apiErr.ErrorResponse)
	}
	f.PrintError(apiErr.Error(), raw)
}

func releaseSnapshot(c *client.Client, path, cursor string, query url.Values) error {
	releaseQuery := cloneValues(query)
	releaseQuery.Del("limit")
	releaseQuery.Del("offset")
	releaseQuery.Del("snapshot")
	releaseQuery.Set("cursor", cursor)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.DeleteQuery(ctx, path, releaseQuery)
	if err != nil {
		return err
	}
	_, err = client.ParseResponse(resp)
	return err
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}
	return clone
}
