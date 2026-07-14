package milestone

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const milestoneReportPageSize = 100

var milestoneNow = time.Now

type milestoneReport struct {
	Repository string                   `json:"repository" yaml:"repository"`
	Milestone  milestoneReportMetadata  `json:"milestone" yaml:"milestone"`
	Summary    milestoneReportSummary   `json:"summary" yaml:"summary"`
	Readiness  milestoneReportReadiness `json:"readiness" yaml:"readiness"`
	Breakdown  milestoneReportBreakdown `json:"breakdown" yaml:"breakdown"`
	Samples    milestoneReportSamples   `json:"samples" yaml:"samples"`
}

type milestoneReportMetadata struct {
	ID            int     `json:"id" yaml:"id"`
	Name          string  `json:"name" yaml:"name"`
	Description   string  `json:"description,omitempty" yaml:"description,omitempty"`
	Status        string  `json:"status" yaml:"status"`
	DueDate       string  `json:"due_date,omitempty" yaml:"due_date,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	CompletionPct float64 `json:"completion_percent" yaml:"completion_percent"`
}

type milestoneReportSummary struct {
	TotalIssues          int  `json:"total_issues" yaml:"total_issues"`
	OpenIssues           int  `json:"open_issues" yaml:"open_issues"`
	ClosedIssues         int  `json:"closed_issues" yaml:"closed_issues"`
	UnassignedOpenIssues int  `json:"unassigned_open_issues" yaml:"unassigned_open_issues"`
	UntaggedOpenIssues   int  `json:"untagged_open_issues" yaml:"untagged_open_issues"`
	CommentedOpenIssues  int  `json:"commented_open_issues" yaml:"commented_open_issues"`
	Overdue              bool `json:"overdue" yaml:"overdue"`
	DaysUntilDue         *int `json:"days_until_due,omitempty" yaml:"days_until_due,omitempty"`
	SampleLimit          int  `json:"sample_limit" yaml:"sample_limit"`
}

type milestoneReportReadiness struct {
	ReadyToClose bool     `json:"ready_to_close" yaml:"ready_to_close"`
	Blockers     []string `json:"blockers,omitempty" yaml:"blockers,omitempty"`
	Warnings     []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type milestoneReportBreakdown struct {
	AllStatuses    []milestoneCountItem `json:"all_statuses" yaml:"all_statuses"`
	OpenPriorities []milestoneCountItem `json:"open_priorities" yaml:"open_priorities"`
	OpenAssignees  []milestoneCountItem `json:"open_assignees" yaml:"open_assignees"`
	OpenTags       []milestoneCountItem `json:"open_tags" yaml:"open_tags"`
}

type milestoneCountItem struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

type milestoneReportSamples struct {
	RecentOpenIssues        []milestoneIssueSample `json:"recent_open_issues" yaml:"recent_open_issues"`
	UnassignedOpenIssues    []milestoneIssueSample `json:"unassigned_open_issues" yaml:"unassigned_open_issues"`
	MostCommentedOpenIssues []milestoneIssueSample `json:"most_commented_open_issues" yaml:"most_commented_open_issues"`
}

type milestoneIssueSample struct {
	Number        int      `json:"number" yaml:"number"`
	DatabaseID    int      `json:"database_id" yaml:"database_id"`
	Title         string   `json:"title" yaml:"title"`
	Status        string   `json:"status,omitempty" yaml:"status,omitempty"`
	Priority      string   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Author        string   `json:"author,omitempty" yaml:"author,omitempty"`
	Assignees     []string `json:"assignees,omitempty" yaml:"assignees,omitempty"`
	Tags          []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	CommentCount  int      `json:"comment_count" yaml:"comment_count"`
	UpdatedAt     string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	UpdatedAtUnix int64    `json:"-" yaml:"-"`
}

type milestoneIssuesPage struct {
	Milestone    map[string]interface{}
	TotalIssues  int
	OpenIssues   int
	ClosedIssues int
	Issues       []map[string]interface{}
}

type milestoneListPage struct {
	TotalCount int
	Items      []map[string]interface{}
}

func newMilestoneReportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "report",
		Description: "Summarize milestone progress, open issue risks, and close readiness",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "Milestone ID"},
			{Name: "name", Short: "n", Usage: "Milestone name"},
			{Name: "sample-limit", Usage: "Number of sample issues to include in each section", Default: "5"},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			report, err := generateMilestoneReport(ctx, ctx.Arg("id"), ctx.Arg("name"), ctx.Arg("sample-limit"))
			if err != nil {
				return err
			}
			return ctx.OutputData(report)
		},
	}
}

func generateMilestoneReport(ctx *common.RuntimeContext, id, name, sampleLimitArg string) (*milestoneReport, error) {
	if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
		return nil, fmt.Errorf("use either --id or --name, not both")
	}
	if strings.TrimSpace(id) == "" && strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("required flag --id or --name is missing")
	}

	sampleLimit, err := parseMilestoneReportSampleLimit(sampleLimitArg)
	if err != nil {
		return nil, err
	}

	milestoneID := strings.TrimSpace(id)
	if milestoneID == "" {
		resolvedID, err := resolveMilestoneIDByName(ctx, strings.TrimSpace(name))
		if err != nil {
			return nil, err
		}
		milestoneID = strconv.Itoa(resolvedID)
	}

	allPage, err := fetchMilestoneIssuesPage(ctx, milestoneID, "", 1, milestoneReportPageSize)
	if err != nil {
		return nil, fmt.Errorf("fetch milestone report: %w", err)
	}
	allIssues, err := fetchAllMilestoneIssues(ctx, milestoneID, "")
	if err != nil {
		return nil, fmt.Errorf("fetch milestone issues: %w", err)
	}
	openIssues, err := fetchAllMilestoneIssues(ctx, milestoneID, "opened")
	if err != nil {
		return nil, fmt.Errorf("fetch open milestone issues: %w", err)
	}

	meta := buildMilestoneReportMetadata(allPage.Milestone)
	report := &milestoneReport{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		Milestone:  meta,
		Summary: buildMilestoneReportSummary(
			meta,
			allPage.TotalIssues,
			allPage.OpenIssues,
			allPage.ClosedIssues,
			openIssues,
			sampleLimit,
		),
		Breakdown: buildMilestoneReportBreakdown(allIssues, openIssues),
		Samples:   buildMilestoneReportSamples(openIssues, sampleLimit),
	}
	report.Readiness = buildMilestoneReportReadiness(report.Milestone, report.Summary)
	return report, nil
}

func parseMilestoneReportSampleLimit(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 5, nil
	}
	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("--sample-limit must be a positive integer")
	}
	return limit, nil
}

func resolveMilestoneIDByName(ctx *common.RuntimeContext, name string) (int, error) {
	milestones, err := fetchAllMilestones(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve milestone by name: %w", err)
	}
	exact := []map[string]interface{}{}
	fuzzy := []map[string]interface{}{}
	target := strings.ToLower(strings.TrimSpace(name))
	for _, milestone := range milestones {
		milestoneName := strings.TrimSpace(stringValue(milestone["name"]))
		if milestoneName == "" {
			continue
		}
		lowerName := strings.ToLower(milestoneName)
		switch {
		case lowerName == target:
			exact = append(exact, milestone)
		case strings.Contains(lowerName, target):
			fuzzy = append(fuzzy, milestone)
		}
	}
	switch {
	case len(exact) == 1:
		return intValue(exact[0]["id"]), nil
	case len(exact) > 1:
		return 0, fmt.Errorf("milestone name %q is ambiguous: %s", name, joinMilestoneMatches(exact))
	case len(fuzzy) == 1:
		return intValue(fuzzy[0]["id"]), nil
	case len(fuzzy) > 1:
		return 0, fmt.Errorf("milestone name %q matched multiple milestones: %s", name, joinMilestoneMatches(fuzzy))
	default:
		return 0, fmt.Errorf("milestone %q not found", name)
	}
}

func fetchAllMilestones(ctx *common.RuntimeContext) ([]map[string]interface{}, error) {
	all := []map[string]interface{}{}
	for page := 1; ; page++ {
		pageData, err := fetchMilestoneListPage(ctx, page, milestoneReportPageSize)
		if err != nil {
			return nil, err
		}
		if len(pageData.Items) == 0 {
			break
		}
		all = append(all, pageData.Items...)
		if pageData.TotalCount > 0 {
			if len(all) >= pageData.TotalCount {
				break
			}
			continue
		}
		if len(pageData.Items) < milestoneReportPageSize {
			break
		}
	}
	return all, nil
}

func fetchMilestoneListPage(ctx *common.RuntimeContext, page, limit int) (*milestoneListPage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	env, err := ctx.CallAPIWithQuery("GET", milestonePath(ctx), q)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("milestone list response did not contain an object")
	}
	return &milestoneListPage{
		TotalCount: firstMilestoneInt(data, "total_count"),
		Items:      objectSlice(data["milestones"]),
	}, nil
}

func fetchAllMilestoneIssues(ctx *common.RuntimeContext, id, category string) ([]map[string]interface{}, error) {
	all := []map[string]interface{}{}
	for page := 1; ; page++ {
		pageData, err := fetchMilestoneIssuesPage(ctx, id, category, page, milestoneReportPageSize)
		if err != nil {
			return nil, err
		}
		if len(pageData.Issues) == 0 {
			break
		}
		all = append(all, pageData.Issues...)
		targetTotal := milestoneIssuesExpectedTotal(pageData, category)
		if targetTotal > 0 {
			if len(all) >= targetTotal {
				break
			}
			continue
		}
		if len(pageData.Issues) < milestoneReportPageSize {
			break
		}
	}
	return all, nil
}

func milestoneIssuesExpectedTotal(pageData *milestoneIssuesPage, category string) int {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "opened", "open", "opening":
		if pageData.OpenIssues > 0 {
			return pageData.OpenIssues
		}
	case "closed", "close":
		if pageData.ClosedIssues > 0 {
			return pageData.ClosedIssues
		}
	}
	return pageData.TotalIssues
}

func fetchMilestoneIssuesPage(ctx *common.RuntimeContext, id, category string, page, limit int) (*milestoneIssuesPage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	setQueryIfPresent(q, "category", category)

	env, err := ctx.CallAPIWithQuery("GET", milestoneItemPath(ctx, id), q)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("milestone view response did not contain an object")
	}

	return &milestoneIssuesPage{
		Milestone:    objectValue(data["milestone"]),
		TotalIssues:  firstMilestoneInt(data, "total_issues_count", "total_count"),
		OpenIssues:   firstMilestoneInt(data, "opened_issues_count", "open_issues_count"),
		ClosedIssues: firstMilestoneInt(data, "closed_issues_count", "close_issues_count"),
		Issues:       objectSlice(data["issues"]),
	}, nil
}

func buildMilestoneReportMetadata(milestone map[string]interface{}) milestoneReportMetadata {
	return milestoneReportMetadata{
		ID:            intValue(milestone["id"]),
		Name:          stringValue(milestone["name"]),
		Description:   stringValue(milestone["description"]),
		Status:        stringValue(milestone["status"]),
		DueDate:       stringValue(milestone["effective_date"]),
		CreatedAt:     stringValue(milestone["created_at"]),
		UpdatedAt:     firstMilestoneString(milestone, "updated_on", "updated_at"),
		CompletionPct: roundMilestonePercent(floatValue(milestone["percent"])),
	}
}

func buildMilestoneReportSummary(meta milestoneReportMetadata, totalIssues, openIssues, closedIssues int, openItems []map[string]interface{}, sampleLimit int) milestoneReportSummary {
	unassigned := 0
	untagged := 0
	commented := 0
	for _, issue := range openItems {
		if len(issueUserNames(issue["assigners"])) == 0 {
			unassigned++
		}
		if len(issueTagNames(issue["tags"])) == 0 {
			untagged++
		}
		if intValue(issue["comment_journals_count"]) > 0 {
			commented++
		}
	}

	summary := milestoneReportSummary{
		TotalIssues:          totalIssues,
		OpenIssues:           openIssues,
		ClosedIssues:         closedIssues,
		UnassignedOpenIssues: unassigned,
		UntaggedOpenIssues:   untagged,
		CommentedOpenIssues:  commented,
		SampleLimit:          sampleLimit,
	}

	if meta.DueDate != "" {
		if dueDate := parseMilestoneTime(meta.DueDate); !dueDate.IsZero() {
			days := milestoneDaysUntil(dueDate)
			summary.DaysUntilDue = &days
			summary.Overdue = days < 0
		}
	}
	return summary
}

func buildMilestoneReportReadiness(meta milestoneReportMetadata, summary milestoneReportSummary) milestoneReportReadiness {
	readiness := milestoneReportReadiness{}
	if strings.EqualFold(meta.Status, "closed") {
		readiness.ReadyToClose = true
		return readiness
	}
	if summary.OpenIssues == 0 {
		readiness.ReadyToClose = true
	} else {
		readiness.Blockers = append(readiness.Blockers, fmt.Sprintf("%d open issues remain", summary.OpenIssues))
	}
	if summary.UnassignedOpenIssues > 0 {
		readiness.Warnings = append(readiness.Warnings, fmt.Sprintf("%d open issues have no assignee", summary.UnassignedOpenIssues))
	}
	if summary.UntaggedOpenIssues > 0 {
		readiness.Warnings = append(readiness.Warnings, fmt.Sprintf("%d open issues have no tags", summary.UntaggedOpenIssues))
	}
	if meta.DueDate == "" {
		readiness.Warnings = append(readiness.Warnings, "milestone has no due date")
	} else if summary.Overdue && summary.DaysUntilDue != nil {
		readiness.Warnings = append(readiness.Warnings, fmt.Sprintf("milestone is overdue by %d days", -(*summary.DaysUntilDue)))
	}
	return readiness
}

func buildMilestoneReportBreakdown(allIssues, openIssues []map[string]interface{}) milestoneReportBreakdown {
	return milestoneReportBreakdown{
		AllStatuses:    countSorted(allIssues, issueStatusName),
		OpenPriorities: countSorted(openIssues, issuePriorityName),
		OpenAssignees:  countSortedMulti(openIssues, func(item map[string]interface{}) []string { return issueUserNames(item["assigners"]) }),
		OpenTags:       countSortedMulti(openIssues, func(item map[string]interface{}) []string { return issueTagNames(item["tags"]) }),
	}
}

func buildMilestoneReportSamples(openIssues []map[string]interface{}, sampleLimit int) milestoneReportSamples {
	recent := buildIssueSamples(openIssues, sampleLimit, func(left, right milestoneIssueSample) bool {
		if left.UpdatedAtUnix != right.UpdatedAtUnix {
			return left.UpdatedAtUnix > right.UpdatedAtUnix
		}
		return left.Number < right.Number
	})
	unassigned := buildIssueSamples(filterMilestoneIssues(openIssues, func(item map[string]interface{}) bool {
		return len(issueUserNames(item["assigners"])) == 0
	}), sampleLimit, func(left, right milestoneIssueSample) bool {
		if left.UpdatedAtUnix != right.UpdatedAtUnix {
			return left.UpdatedAtUnix > right.UpdatedAtUnix
		}
		return left.Number < right.Number
	})
	commented := buildIssueSamples(filterMilestoneIssues(openIssues, func(item map[string]interface{}) bool {
		return intValue(item["comment_journals_count"]) > 0
	}), sampleLimit, func(left, right milestoneIssueSample) bool {
		if left.CommentCount != right.CommentCount {
			return left.CommentCount > right.CommentCount
		}
		if left.UpdatedAtUnix != right.UpdatedAtUnix {
			return left.UpdatedAtUnix > right.UpdatedAtUnix
		}
		return left.Number < right.Number
	})

	return milestoneReportSamples{
		RecentOpenIssues:        recent,
		UnassignedOpenIssues:    unassigned,
		MostCommentedOpenIssues: commented,
	}
}

func buildIssueSamples(items []map[string]interface{}, sampleLimit int, less func(left, right milestoneIssueSample) bool) []milestoneIssueSample {
	samples := make([]milestoneIssueSample, 0, len(items))
	for _, item := range items {
		samples = append(samples, normalizeMilestoneIssueSample(item))
	}
	sort.Slice(samples, func(i, j int) bool {
		return less(samples[i], samples[j])
	})
	if len(samples) > sampleLimit {
		samples = samples[:sampleLimit]
	}
	for i := range samples {
		samples[i].UpdatedAtUnix = 0
	}
	return samples
}

func normalizeMilestoneIssueSample(item map[string]interface{}) milestoneIssueSample {
	updatedAt := firstMilestoneString(item, "updated_at", "updated_on")
	updatedAtUnix := int64(0)
	if parsed := parseMilestoneTime(updatedAt); !parsed.IsZero() {
		updatedAtUnix = parsed.Unix()
	}
	return milestoneIssueSample{
		Number:        firstMilestoneInt(item, "project_issues_index", "number"),
		DatabaseID:    intValue(item["id"]),
		Title:         firstMilestoneString(item, "subject", "title"),
		Status:        issueStatusName(item),
		Priority:      issuePriorityName(item),
		Author:        userDisplayName(objectValue(item["author"])),
		Assignees:     issueUserNames(item["assigners"]),
		Tags:          issueTagNames(item["tags"]),
		CommentCount:  intValue(item["comment_journals_count"]),
		UpdatedAt:     updatedAt,
		UpdatedAtUnix: updatedAtUnix,
	}
}

func filterMilestoneIssues(items []map[string]interface{}, keep func(item map[string]interface{}) bool) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func countSorted(items []map[string]interface{}, name func(item map[string]interface{}) string) []milestoneCountItem {
	counts := map[string]int{}
	for _, item := range items {
		key := strings.TrimSpace(name(item))
		if key == "" {
			key = "Unspecified"
		}
		counts[key]++
	}
	return sortMilestoneCounts(counts)
}

func countSortedMulti(items []map[string]interface{}, names func(item map[string]interface{}) []string) []milestoneCountItem {
	counts := map[string]int{}
	for _, item := range items {
		values := names(item)
		if len(values) == 0 {
			counts["Unspecified"]++
			continue
		}
		for _, value := range values {
			key := strings.TrimSpace(value)
			if key == "" {
				key = "Unspecified"
			}
			counts[key]++
		}
	}
	return sortMilestoneCounts(counts)
}

func sortMilestoneCounts(counts map[string]int) []milestoneCountItem {
	items := make([]milestoneCountItem, 0, len(counts))
	for name, count := range counts {
		items = append(items, milestoneCountItem{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func issueStatusName(item map[string]interface{}) string {
	if name := firstMilestoneString(item, "status_name"); name != "" {
		return name
	}
	return firstMilestoneString(objectValue(item["status"]), "name")
}

func issuePriorityName(item map[string]interface{}) string {
	if name := firstMilestoneString(item, "priority_name"); name != "" {
		return name
	}
	return firstMilestoneString(objectValue(item["priority"]), "name")
}

func issueUserNames(value interface{}) []string {
	users := objectSlice(value)
	names := make([]string, 0, len(users))
	for _, user := range users {
		if name := userDisplayName(user); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func userDisplayName(user map[string]interface{}) string {
	if name := strings.TrimSpace(stringValue(user["name"])); name != "" {
		return name
	}
	return strings.TrimSpace(stringValue(user["login"]))
}

func issueTagNames(value interface{}) []string {
	tags := objectSlice(value)
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		if name := strings.TrimSpace(stringValue(tag["name"])); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func milestoneDaysUntil(dueDate time.Time) int {
	now := milestoneNow()
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	due := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, now.Location())
	return int(math.Round(due.Sub(nowDate).Hours() / 24))
}

func parseMilestoneTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, trimmed, time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func roundMilestonePercent(value float64) float64 {
	return math.Round(value*10000) / 100
}

func joinMilestoneMatches(items []map[string]interface{}) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%d:%s", intValue(item["id"]), stringValue(item["name"])))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func firstMilestoneString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringValue(item[key])); value != "" {
			return value
		}
	}
	return ""
}

func firstMilestoneInt(item map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return intValue(value)
		}
	}
	return 0
}

func objectValue(value interface{}) map[string]interface{} {
	item, _ := value.(map[string]interface{})
	return item
}

func objectSlice(value interface{}) []map[string]interface{} {
	switch typed := value.(type) {
	case []map[string]interface{}:
		return append([]map[string]interface{}(nil), typed...)
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, raw := range typed {
			if item, ok := raw.(map[string]interface{}); ok {
				items = append(items, item)
			}
		}
		return items
	default:
		return nil
	}
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	default:
		return 0
	}
}

func floatValue(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return n
	default:
		return 0
	}
}
