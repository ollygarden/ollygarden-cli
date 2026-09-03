package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	servicesGroupedLimit          int
	servicesGroupedOffset         int
	servicesGroupedSort           string
	servicesGroupedQuery          string
	servicesGroupedView           string
	servicesGroupedEnvironment    string
	servicesGroupedMinScore       int
	servicesGroupedMaxScore       int
	servicesGroupedHasInsightType string
	servicesGroupedOrder          string
	servicesGroupedSnapshot       snapshotFlags
)

var allowedSorts = map[string]bool{
	"insights-first": true,
	"name-asc":       true,
	"name-desc":      true,
	"created-asc":    true,
	"created-desc":   true,
	"score":          true,
	"name":           true,
	"insight_count":  true,
	"last_seen":      true,
}

type groupedServiceItem struct {
	Name                 string               `json:"name"`
	Environment          string               `json:"environment"`
	Namespace            string               `json:"namespace"`
	LatestID             string               `json:"latest_id"`
	VersionCount         int                  `json:"version_count"`
	InsightsCount        int                  `json:"insights_count"`
	InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
	Environments         []string             `json:"environments"`
	LastSeenAt           string               `json:"last_seen_at"`
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
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedSort, "sort", "insights-first", "Sort: insights-first, name-asc, name-desc, created-asc, created-desc; service view: score, name, insight_count, last_seen")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedQuery, "query", "", "Search service names")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedView, "view", "", "Result view (service requires snapshot pagination)")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedEnvironment, "environment", "", "Service view: restrict facts to one environment")
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedMinScore, "min-score", 0, "Service view: minimum instrumentation score (0-100)")
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedMaxScore, "max-score", 0, "Service view: maximum instrumentation score (0-100)")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedHasInsightType, "has-insight-type", "", "Service view: require an active insight of this exact type")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedOrder, "order", "", "Service view sort direction (asc, desc)")
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
	effectiveSort := servicesGroupedSort
	if servicesGroupedView == "service" && !cmd.Flags().Changed("sort") {
		effectiveSort = "score"
	}
	if err := validateServicesGroupedIdentityFlags(cmd, effectiveSort); err != nil {
		return err
	}
	if servicesGroupedView != "service" && (servicesGroupedSnapshot.enabled || servicesGroupedSnapshot.cursor != "" || servicesGroupedSnapshot.all) {
		return fmt.Errorf("--snapshot, --cursor, and --all require --view service")
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
	query.Set("sort", effectiveSort)
	if servicesGroupedQuery != "" {
		query.Set("q", servicesGroupedQuery)
	}
	if servicesGroupedView != "" {
		query.Set("view", servicesGroupedView)
	}
	if servicesGroupedEnvironment != "" {
		query.Set("environment", servicesGroupedEnvironment)
	}
	if cmd.Flags().Changed("min-score") {
		query.Set("min_score", strconv.Itoa(servicesGroupedMinScore))
	}
	if cmd.Flags().Changed("max-score") {
		query.Set("max_score", strconv.Itoa(servicesGroupedMaxScore))
	}
	if servicesGroupedHasInsightType != "" {
		query.Set("has_insight_type", servicesGroupedHasInsightType)
	}
	if servicesGroupedOrder != "" {
		query.Set("order", servicesGroupedOrder)
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
		openCursor = apiResp.Meta.NextCursor
		var pageItems []groupedServiceItem
		if err := json.Unmarshal(apiResp.Data, &pageItems); err != nil {
			return fmt.Errorf("parsing grouped services data: %w", err)
		}
		services = append(services, pageItems...)
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
	if servicesGroupedView == "service" {
		headers = []string{"NAME", "ENVIRONMENTS", "INSIGHTS", "SCORE", "LAST SEEN"}
	}
	rows := make([][]string, len(services))
	for i, s := range services {
		score := "\u2014"
		if s.InstrumentationScore != nil {
			score = strconv.Itoa(s.InstrumentationScore.Score)
		}
		if servicesGroupedView == "service" {
			rows[i] = []string{s.Name, strings.Join(s.Environments, ", "), strconv.Itoa(s.InsightsCount), score, s.LastSeenAt}
		} else {
			rows[i] = []string{s.Name, s.Environment, strconv.Itoa(s.VersionCount), strconv.Itoa(s.InsightsCount), score}
		}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.NextCursor != "" && !servicesGroupedSnapshot.all {
		fmt.Fprintf(cmd.ErrOrStderr(), "# More results. Continue with --cursor %q and the same view, filters, and sort.\n", apiResp.Meta.NextCursor)
	} else if apiResp.Meta.HasMore && !servicesGroupedSnapshot.all {
		f.PrintPaginationHint(apiResp.Meta.Total, servicesGroupedOffset, servicesGroupedLimit)
	}

	return nil
}

func validateServicesGroupedIdentityFlags(cmd *cobra.Command, effectiveSort string) error {
	if servicesGroupedView != "" && servicesGroupedView != "service" {
		return fmt.Errorf("--view must be service")
	}
	if cmd.Flags().Changed("min-score") && (servicesGroupedMinScore < 0 || servicesGroupedMinScore > 100) {
		return fmt.Errorf("--min-score must be between 0 and 100")
	}
	if cmd.Flags().Changed("max-score") && (servicesGroupedMaxScore < 0 || servicesGroupedMaxScore > 100) {
		return fmt.Errorf("--max-score must be between 0 and 100")
	}
	if cmd.Flags().Changed("min-score") && cmd.Flags().Changed("max-score") && servicesGroupedMinScore > servicesGroupedMaxScore {
		return fmt.Errorf("--min-score must not exceed --max-score")
	}
	if servicesGroupedOrder != "" && servicesGroupedOrder != "asc" && servicesGroupedOrder != "desc" {
		return fmt.Errorf("--order must be one of: asc, desc")
	}
	for flag, value := range map[string]string{
		"--query":            servicesGroupedQuery,
		"--environment":      servicesGroupedEnvironment,
		"--has-insight-type": servicesGroupedHasInsightType,
	} {
		if cmd.Flags().Changed(strings.TrimPrefix(flag, "--")) {
			length := len([]rune(strings.TrimSpace(value)))
			if length < 1 || length > 128 {
				return fmt.Errorf("%s must be between 1 and 128 characters", flag)
			}
		}
	}
	identityFlag := servicesGroupedEnvironment != "" || cmd.Flags().Changed("min-score") || cmd.Flags().Changed("max-score") || servicesGroupedHasInsightType != "" || servicesGroupedOrder != ""
	serviceSort := effectiveSort == "score" || effectiveSort == "name" || effectiveSort == "insight_count" || effectiveSort == "last_seen"
	if (identityFlag || serviceSort) && servicesGroupedView != "service" {
		return fmt.Errorf("--view service is required with service-identity filters, --order, or service-identity sort fields")
	}
	if servicesGroupedView == "service" && cmd.Flags().Changed("sort") && !serviceSort {
		return fmt.Errorf("--view service requires --sort to be one of: score, name, insight_count, last_seen")
	}
	if servicesGroupedView != "service" && !allowedSorts[effectiveSort] {
		return fmt.Errorf("--sort must be one of: insights-first, name-asc, name-desc, created-asc, created-desc")
	}
	return nil
}
