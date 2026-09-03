package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	servicesGroupedLimit    int
	servicesGroupedOffset   int
	servicesGroupedSort     string
	servicesGroupedView     string
	servicesGroupedSnapshot snapshotFlags
)

var allowedSorts = map[string]bool{
	"insights-first": true,
	"name-asc":       true,
	"name-desc":      true,
	"created-asc":    true,
	"created-desc":   true,
}

type groupedServiceItem struct {
	Name                 string               `json:"name"`
	Environment          string               `json:"environment"`
	Namespace            string               `json:"namespace"`
	LatestID             string               `json:"latest_id"`
	VersionCount         int                  `json:"version_count"`
	InsightsCount        int                  `json:"insights_count"`
	InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
}

var servicesGroupedCmd = &cobra.Command{
	Use:   "grouped",
	Short: "List services grouped by name",
	Args:  cobra.NoArgs,
	RunE:  runServicesGrouped,
}

func init() {
	servicesCmd.AddCommand(servicesGroupedCmd)
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedLimit, "limit", 50, "Maximum number of results (1-100)")
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedOffset, "offset", 0, "Number of results to skip (≥0)")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedSort, "sort", "insights-first", "Sort order: insights-first, name-asc, name-desc, created-asc, created-desc")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedView, "view", "", "Grouped view (service requires snapshot pagination)")
	servicesGroupedCmd.Flags().BoolVar(&servicesGroupedSnapshot.enabled, "snapshot", false, "Start a mutation-stable snapshot")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedSnapshot.cursor, "cursor", "", "Continue with an opaque snapshot cursor")
	servicesGroupedCmd.Flags().BoolVar(&servicesGroupedSnapshot.all, "all", false, "Read all pages from one mutation-stable snapshot")
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedSnapshot.maxPages, "max-pages", 0, "Stop --all after this many pages and release the snapshot (0 means unlimited)")
}

func runServicesGrouped(cmd *cobra.Command, args []string) error {
	if servicesGroupedLimit < 1 || servicesGroupedLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if servicesGroupedOffset < 0 {
		return fmt.Errorf("--offset must be >= 0")
	}
	if servicesGroupedView != "" && servicesGroupedView != "service" {
		return fmt.Errorf("--view must be service")
	}
	if servicesGroupedView != "service" && (servicesGroupedSnapshot.enabled || servicesGroupedSnapshot.cursor != "" || servicesGroupedSnapshot.all) {
		return fmt.Errorf("--snapshot, --cursor, and --all require --view service")
	}
	if servicesGroupedView == "service" && !cmd.Flags().Changed("sort") {
		servicesGroupedSort = "score"
	}
	if servicesGroupedView == "service" {
		if servicesGroupedSort != "score" && servicesGroupedSort != "name" && servicesGroupedSort != "insight_count" && servicesGroupedSort != "last_seen" {
			return fmt.Errorf("--sort with --view service must be one of: score, name, insight_count, last_seen")
		}
	} else if !allowedSorts[servicesGroupedSort] {
		return fmt.Errorf("--sort must be one of: insights-first, name-asc, name-desc, created-asc, created-desc")
	}
	if servicesGroupedView == "service" {
		servicesGroupedSnapshot.enabled = true
	}
	if err := servicesGroupedSnapshot.validate(servicesGroupedOffset, jsonMode); err != nil {
		return err
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(servicesGroupedLimit))
	query.Set("offset", strconv.Itoa(servicesGroupedOffset))
	query.Set("sort", servicesGroupedSort)
	if servicesGroupedView != "" {
		query.Set("view", servicesGroupedView)
	}
	servicesGroupedSnapshot.apply(query)

	var services []groupedServiceItem
	var apiResp *client.APIResponse
	openCursor := ""
	if servicesGroupedSnapshot.all {
		defer func() {
			if openCursor != "" {
				if err := releaseSnapshot(c, "/services/grouped/snapshot", openCursor, query); err != nil && !quiet {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not release unfinished services snapshot: %v\n", err)
				}
			}
		}()
	}
	for page := 1; ; page++ {
		resp, err := c.Get(cmd.Context(), "/services/grouped", query)
		if err != nil {
			return fmt.Errorf("requesting grouped services: %w", err)
		}
		apiResp, err = client.ParseResponse(resp)
		if err != nil {
			printAPIError(f, err)
			return err
		}
		if !servicesGroupedSnapshot.all {
			break
		}
		var pageItems []groupedServiceItem
		if err := json.Unmarshal(apiResp.Data, &pageItems); err != nil {
			return fmt.Errorf("parsing grouped services data: %w", err)
		}
		services = append(services, pageItems...)
		openCursor = apiResp.Meta.NextCursor
		if openCursor == "" {
			break
		}
		if servicesGroupedSnapshot.maxPages > 0 && page >= servicesGroupedSnapshot.maxPages {
			break
		}
		query.Set("cursor", openCursor)
	}

	if f.IsJSON() {
		raw, _ := json.Marshal(apiResp)
		f.PrintJSON(raw)
		return nil
	}

	if f.IsQuiet() {
		return nil
	}

	if !servicesGroupedSnapshot.all {
		if err := json.Unmarshal(apiResp.Data, &services); err != nil {
			return fmt.Errorf("parsing grouped services data: %w", err)
		}
	}

	headers := []string{"NAME", "ENVIRONMENT", "VERSIONS", "INSIGHTS", "SCORE"}
	rows := make([][]string, len(services))
	for i, s := range services {
		score := "\u2014"
		if s.InstrumentationScore != nil {
			score = strconv.Itoa(s.InstrumentationScore.Score)
		}
		rows[i] = []string{s.Name, s.Environment, strconv.Itoa(s.VersionCount), strconv.Itoa(s.InsightsCount), score}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.NextCursor != "" && !servicesGroupedSnapshot.all {
		fmt.Fprintf(cmd.ErrOrStderr(), "# More results. Continue with --cursor %q and the same view, filters, and sort.\n", apiResp.Meta.NextCursor)
	} else if apiResp.Meta.HasMore && !servicesGroupedSnapshot.all {
		f.PrintPaginationHint(apiResp.Meta.Total, servicesGroupedOffset, servicesGroupedLimit)
	}

	return nil
}
