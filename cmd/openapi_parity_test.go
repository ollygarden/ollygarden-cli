package cmd

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate sh -c "curl -fsSL https://api.ollygarden.cloud/openapi.json -o testdata/olive-openapi.json"

// knownDifferences lists "command --flag" pairs whose CLI values differ from
// the document on purpose. Each entry needs a one-line reason.
var knownDifferences = map[string]string{}

type openAPIOperation struct {
	Parameters []struct {
		Name   string `json:"name"`
		In     string `json:"in"`
		Schema struct {
			Enum []string `json:"enum"`
		} `json:"schema"`
	} `json:"parameters"`
}

type openAPIDocument struct {
	Paths map[string]struct {
		Get *openAPIOperation `json:"get"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		} `json:"schemas"`
	} `json:"components"`
}

func (d openAPIDocument) queryEnum(path, param string) []string {
	item, ok := d.Paths[path]
	if !ok || item.Get == nil {
		return nil
	}
	for _, p := range item.Get.Parameters {
		if p.In == "query" && p.Name == param {
			return p.Schema.Enum
		}
	}
	return nil
}

func (d openAPIDocument) bodyEnum(schema, property string) []string {
	return d.Components.Schemas[schema].Properties[property].Enum
}

// TestFlagValuesMatchOpenAPI fails when a flag's allowed values differ from
// the enum Olive declares for the same parameter. Parameters the document
// describes only in prose have no enum and are not checked.
func TestFlagValuesMatchOpenAPI(t *testing.T) {
	raw, err := os.ReadFile("testdata/olive-openapi.json")
	require.NoError(t, err)
	var doc openAPIDocument
	require.NoError(t, json.Unmarshal(raw, &doc))

	tests := []struct {
		command string
		flag    string
		cli     []string
		olive   []string
	}{
		{"analytics log-volume", "--period", slices.Collect(maps.Keys(validLogVolumePeriods)), doc.queryEnum("/v1/analytics/log-volume", "period")},
		{"services grouped", "--sort", slices.Collect(maps.Keys(allowedSorts)), doc.queryEnum("/v1/services/grouped", "sort")},
		{"services grouped", "--order", servicesGroupedOrders, doc.queryEnum("/v1/services/grouped", "order")},
		{"rose executions findings", "--status", roseFindingStatuses, doc.queryEnum("/v1/rose/codebase/findings", "status")},
		{"rose executions findings", "--dismissed", roseDismissedFilters, doc.queryEnum("/v1/rose/codebase/findings", "dismissed")},
		{"rose findings list", "--dismissed", roseDismissedFilters, doc.queryEnum("/v1/rose/findings", "dismissed")},
		{"rose executions instrument", "--type", roseInstrumentationTypes, doc.bodyEnum("models.RoseInstrumentationExecutionRequest", "type")},
		{"webhooks create", "--min-severity", slices.Collect(maps.Keys(validSeverities)), doc.bodyEnum("models.CreateWebhookRequest", "min_severity")},
		{"webhooks update", "--min-severity", slices.Collect(maps.Keys(validSeverities)), doc.bodyEnum("models.UpdateWebhookRequest", "min_severity")},
	}

	for _, tt := range tests {
		name := tt.command + " " + tt.flag
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, tt.olive, "%s: the document declares no enum for this parameter; remove it from this test", name)

			onlyCLI, onlyOlive := difference(tt.cli, tt.olive), difference(tt.olive, tt.cli)
			differs := len(onlyCLI) > 0 || len(onlyOlive) > 0

			reason, excluded := knownDifferences[name]
			if excluded {
				require.True(t, differs, "%s is in knownDifferences (%s) but now matches; remove the entry", name, reason)
				return
			}
			require.False(t, differs, "%s: CLI accepts values olive does not: %v; olive accepts values the CLI does not: %v", name, onlyCLI, onlyOlive)
		})
	}
}

// difference returns the values in a that are not in b.
func difference(a, b []string) []string {
	var out []string
	for _, v := range a {
		if !slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	slices.Sort(out)
	return out
}
