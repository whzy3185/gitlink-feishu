package workflow

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func FetchHealthInput(ctx *common.RuntimeContext, opts HealthFetchOptions) (HealthInput, []ScoringNote, error) {
	owner, repo, err := resolveFetchRepo(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return HealthInput{}, nil, err
	}

	input := HealthInput{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
	}
	notes := []ScoringNote{}
	staleDays := opts.StaleDays
	if staleDays <= 0 {
		staleDays = 30
	}

	repoInfo, repoErr := fetchRepoInfo(ctx, owner, repo)
	if repoErr != nil {
		notes = append(notes, ScoringNote{Metric: "repository", Note: repoErr.Error()})
	} else {
		applyRepoSignals(&input, repoInfo)
	}

	if issues, err := fetchAllListItems(ctx, workflowRepoPath(owner, repo)+"/issues", issueListQuery("open"), 100); err != nil {
		notes = append(notes, ScoringNote{Metric: "open_issues", Note: fmt.Sprintf("issue probe failed: %v", err)})
	} else {
		input.OpenIssues = len(issues)
		input.StaleIssues = countStaleItems(issues, staleDays)
		input.RecentActivityKnown, input.RecentActivityDays, input = updateRecentActivity(input, latestTimeFromItems(issues, input.RecentActivityDays))
	}

	if prs, err := fetchAllListItems(ctx, workflowRepoPath(owner, repo)+"/pulls", issueListQuery("open"), 100); err != nil {
		notes = append(notes, ScoringNote{Metric: "open_prs", Note: fmt.Sprintf("pull request probe failed: %v", err)})
	} else {
		input.OpenPRs = len(prs)
		input.StalePRs = countStaleItems(prs, staleDays)
		input.RecentActivityKnown, input.RecentActivityDays, input = updateRecentActivity(input, latestTimeFromItems(prs, input.RecentActivityDays))
	}

	if opts.IncludeRelease {
		if releases, err := fetchAllListItems(ctx, ctx.RepoPath()+"/releases", nil, 100); err != nil {
			input.ReleaseKnown = false
			notes = append(notes, ScoringNote{Metric: "release_status", Note: fmt.Sprintf("release probe failed: %v", err)})
		} else {
			input.ReleaseKnown = true
			input.HasRecentRelease = len(releases) > 0
			input.RecentActivityKnown, input.RecentActivityDays, input = updateRecentActivity(input, latestTimeFromItems(releases, input.RecentActivityDays))
		}
	}

	if opts.IncludeCI {
		if builds, err := fetchAllListItems(ctx, ctx.RepoPath()+"/builds", queryWithPageLimit(nil, 1, 20), 20); err != nil {
			input.CIKnown = false
			notes = append(notes, ScoringNote{Metric: "ci_status", Note: fmt.Sprintf("ci probe failed: %v", err)})
		} else {
			input.CIKnown = true
			input.CIPassing = len(builds) > 0 && buildPassing(builds[0])
		}
	}

	if opts.IncludeDocs {
		applyDocSignals(&input, repoInfo, &notes)
	}

	applyAgentReadinessEstimate(&input)

	if !input.RecentActivityKnown {
		notes = append(notes, ScoringNote{Metric: "recent_activity", Note: "recent activity unavailable; scored conservatively"})
	}
	if !input.ReleaseKnown {
		notes = append(notes, ScoringNote{Metric: "release_status", Note: "release status unavailable; scored conservatively"})
	}
	if !input.CIKnown {
		notes = append(notes, ScoringNote{Metric: "ci_status", Note: "ci status unavailable; scored conservatively"})
	}

	return input, uniqueScoringNotes(notes), nil
}

func fetchRepoInfo(ctx *common.RuntimeContext, owner, repo string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", workflowRepoPath(owner, repo), nil)
	if err != nil {
		return nil, err
	}
	info := apiObject(env.Data)
	if info == nil {
		return nil, fmt.Errorf("repository response did not contain an object")
	}
	return info, nil
}

func applyRepoSignals(input *HealthInput, repoInfo map[string]interface{}) {
	if repoInfo == nil {
		return
	}
	if t := apiTime(repoInfo["updated_at"]); !t.IsZero() {
		input.RecentActivityKnown = true
		input.RecentActivityDays = apiAgeInDays(t)
	}
	applyDocSignals(input, repoInfo, nil)
}

func applyDocSignals(input *HealthInput, repoInfo map[string]interface{}, notes *[]ScoringNote) {
	if repoInfo == nil {
		return
	}
	hasReadme, readmeOK := repoInfo["has_readme"]
	hasLicense, licenseOK := repoInfo["has_license"]
	hasContributing, contribOK := repoInfo["has_contributing"]
	if readmeOK {
		input.HasReadme = apiBool(hasReadme)
	} else if notes != nil {
		*notes = append(*notes, ScoringNote{Metric: "documentation", Note: "README probe unavailable; scored conservatively"})
	}
	if licenseOK {
		input.HasLicense = apiBool(hasLicense)
	} else if notes != nil {
		*notes = append(*notes, ScoringNote{Metric: "license_status", Note: "license probe unavailable; scored conservatively"})
	}
	if contribOK {
		input.HasContributing = apiBool(hasContributing)
	} else if notes != nil {
		*notes = append(*notes, ScoringNote{Metric: "contribution_status", Note: "contributing probe unavailable; scored conservatively"})
	}
}

func applyAgentReadinessEstimate(input *HealthInput) {
	score := 4
	if input.HasReadme {
		score += 2
	}
	if input.HasLicense {
		score += 2
	}
	if input.HasContributing {
		score += 2
	}
	if input.RecentActivityKnown {
		score++
	}
	if input.ReleaseKnown {
		score++
	}
	input.AgentReadinessKnown = true
	input.AgentReadinessScore = clampInt(score, 0, 10)
}

func countStaleItems(items []map[string]interface{}, staleDays int) int {
	if staleDays <= 0 {
		staleDays = 30
	}
	count := 0
	for _, item := range items {
		if apiAgeInDays(itemActivityTime(item)) >= staleDays {
			count++
		}
	}
	return count
}

func itemActivityTime(item map[string]interface{}) time.Time {
	if item == nil {
		return time.Time{}
	}
	return apiLatestTime(
		apiTime(item["updated_at"]),
		apiTime(item["updatedAt"]),
		apiTime(item["created_at"]),
		apiTime(item["last_updated_at"]),
		apiTime(item["lastUpdatedAt"]),
		apiTime(item["last_activity_at"]),
		apiTime(item["lastActivityAt"]),
		apiTime(item["merged_at"]),
		apiTime(item["mergedAt"]),
		apiTime(item["closed_at"]),
		apiTime(item["closedAt"]),
	)
}

func latestTimeFromItems(items []map[string]interface{}, currentDays int) time.Time {
	latest := time.Time{}
	for _, item := range items {
		latest = apiLatestTime(latest, itemActivityTime(item))
	}
	return latest
}

func updateRecentActivity(input HealthInput, latest time.Time) (bool, int, HealthInput) {
	if latest.IsZero() {
		return input.RecentActivityKnown, input.RecentActivityDays, input
	}
	days := apiAgeInDays(latest)
	if !input.RecentActivityKnown || days < input.RecentActivityDays || input.RecentActivityDays == 0 {
		input.RecentActivityKnown = true
		input.RecentActivityDays = days
	}
	return input.RecentActivityKnown, input.RecentActivityDays, input
}

func queryWithPageLimit(base url.Values, page, limit int) url.Values {
	if base == nil {
		base = url.Values{}
	}
	if page > 0 {
		base.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		base.Set("limit", fmt.Sprintf("%d", limit))
	}
	return base
}

func issueListQuery(state string) url.Values {
	q := url.Values{}
	q.Set("state", state)
	return q
}

func fetchAllListItems(ctx *common.RuntimeContext, path string, baseQuery url.Values, pageSize int) ([]map[string]interface{}, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	all := []map[string]interface{}{}
	for page := 1; ; page++ {
		query := cloneValues(baseQuery)
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("limit", fmt.Sprintf("%d", pageSize))

		env, err := ctx.CallAPIWithQuery("GET", path, query)
		if err != nil {
			return nil, err
		}
		items := apiList(env.Data)
		pageItems := make([]map[string]interface{}, 0, len(items))
		for _, raw := range items {
			if item, ok := raw.(map[string]interface{}); ok {
				pageItems = append(pageItems, item)
			}
		}
		if len(pageItems) == 0 {
			break
		}
		all = append(all, pageItems...)
		if len(pageItems) < pageSize {
			break
		}
	}
	return all, nil
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	out := url.Values{}
	for key, list := range values {
		out[key] = append([]string(nil), list...)
	}
	return out
}

func buildPassing(item map[string]interface{}) bool {
	for _, key := range []string{"status", "state", "result", "conclusion", "status_text"} {
		if value := strings.ToLower(strings.TrimSpace(apiString(item[key]))); value != "" {
			switch value {
			case "success", "passed", "pass", "ok", "done", "succeeded", "build passed":
				return true
			case "failed", "failure", "error", "canceled", "cancelled", "running", "pending":
				return false
			}
		}
	}
	if apiBool(item["success"]) {
		return true
	}
	return false
}

func uniqueScoringNotes(notes []ScoringNote) []ScoringNote {
	seen := map[string]struct{}{}
	out := make([]ScoringNote, 0, len(notes))
	for _, note := range notes {
		key := note.Metric + "|" + note.Note
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, note)
	}
	return out
}
