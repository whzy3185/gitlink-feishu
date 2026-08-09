package feishu

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func reviewDataObject(value interface{}) map[string]interface{} {
	value = normalizeReviewDataJSON(value)
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"pull_request", "repository", "item", "record", "data"} {
			if nested, ok := typed[key].(map[string]interface{}); ok {
				return nested
			}
		}
		return typed
	case []interface{}:
		if len(typed) == 1 {
			if item, ok := typed[0].(map[string]interface{}); ok {
				return item
			}
		}
	}
	return nil
}

func reviewDataList(value interface{}) []map[string]interface{} {
	value = normalizeReviewDataJSON(value)
	var raw []interface{}
	switch typed := value.(type) {
	case []interface{}:
		raw = typed
	case map[string]interface{}:
		for _, key := range []string{"pulls", "reviews", "journals", "issue_journals", "files", "versions", "items", "records", "data"} {
			if nested := reviewDataList(typed[key]); len(nested) > 0 {
				return nested
			}
		}
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if object, ok := item.(map[string]interface{}); ok {
			out = append(out, object)
		}
	}
	return out
}

func normalizeReviewDataJSON(value interface{}) interface{} {
	switch typed := value.(type) {
	case nil, map[string]interface{}, []interface{}:
		return typed
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
			return typed
		}
		var decoded interface{}
		decoder := json.NewDecoder(strings.NewReader(trimmed))
		decoder.UseNumber()
		if decoder.Decode(&decoded) == nil {
			return decoded
		}
	}
	return value
}

func populateReviewDataPR(data *ReviewData, item map[string]interface{}) {
	data.Title = firstReviewDataString(item, "title", "subject")
	data.Author = firstReviewDataActor(item, "create_user", "fork_project_user", "author", "user")
	data.State = normalizeReviewDataPRState(firstReviewDataString(item, "state", "pull_request_staus", "status"))
	if merged, ok := reviewDataBool(item, "merged"); ok && merged {
		data.State = "merged"
	}
	data.BaseBranch = firstReviewDataBranch(item, "base_branch", "target_branch", "base")
	data.HeadBranch = firstReviewDataBranch(item, "head_branch", "source_branch", "head")
	data.HeadSHA = firstReviewDataString(item, "head_commit_sha", "head_sha", "commit_sha")
	if head, ok := item["head"].(map[string]interface{}); ok {
		data.HeadSHA = firstReviewDataValue(data.HeadSHA, firstReviewDataString(head, "sha", "commit_id", "head_commit_sha"))
		data.HeadBranch = firstReviewDataValue(data.HeadBranch, firstReviewDataString(head, "ref", "branch", "name"))
	}
	data.VersionID = firstReviewDataString(item, "version_id", "current_version_id", "patchset_id")
	data.GitLinkURL = firstReviewDataString(item, "html_url", "web_url", "url")
	if data.GitLinkURL == "" {
		data.GitLinkURL = fmt.Sprintf("https://www.gitlink.org.cn/%s/pulls/%d", data.Repository, data.PullRequest)
	}
	data.Patchset = ReviewDataPatchset{
		ID:           data.VersionID,
		HeadSHA:      data.HeadSHA,
		FilesCount:   firstReviewDataInt(item, "files_count", "changed_files"),
		CommitsCount: firstReviewDataInt(item, "commits_count", "commits"),
		Additions:    firstReviewDataInt(item, "add_line_num", "additions"),
		Deletions:    firstReviewDataInt(item, "del_line_num", "deletions"),
	}
}

func normalizeReviewDataPatchset(items []map[string]interface{}, fallback ReviewDataPatchset) ReviewDataPatchset {
	current := fallback
	for _, item := range items {
		candidate := ReviewDataPatchset{
			ID:           firstReviewDataString(item, "id", "version_id"),
			HeadSHA:      firstReviewDataString(item, "head_commit_sha", "head_sha", "commit_sha"),
			BaseSHA:      firstReviewDataString(item, "base_commit_sha", "base_sha"),
			StartSHA:     firstReviewDataString(item, "start_commit_sha", "start_sha"),
			FilesCount:   firstReviewDataInt(item, "files_count", "changed_files"),
			CommitsCount: firstReviewDataInt(item, "commits_count", "commits"),
			Additions:    firstReviewDataInt(item, "add_line_num", "additions"),
			Deletions:    firstReviewDataInt(item, "del_line_num", "deletions"),
			CreatedAt:    formatReviewDataTime(firstReviewDataTime(item, "created_time", "created_at")),
			UpdatedAt:    formatReviewDataTime(firstReviewDataTime(item, "updated_time", "updated_at")),
		}
		if current.ID == "" || reviewDataPatchsetLater(candidate, current) {
			current = candidate
		}
	}
	return current
}

func reviewDataPatchsetLater(left, right ReviewDataPatchset) bool {
	leftTime := firstReviewDataParsedTime(left.UpdatedAt, left.CreatedAt)
	rightTime := firstReviewDataParsedTime(right.UpdatedAt, right.CreatedAt)
	if !leftTime.IsZero() && !rightTime.IsZero() && !leftTime.Equal(rightTime) {
		return leftTime.After(rightTime)
	}
	leftID, leftOK := reviewDataNumericID(left.ID)
	rightID, rightOK := reviewDataNumericID(right.ID)
	if leftOK && rightOK {
		return leftID > rightID
	}
	return left.ID > right.ID
}

func normalizeReviewDataReviews(items []map[string]interface{}, headSHA string) []ReviewDataReview {
	out := make([]ReviewDataReview, 0, len(items))
	for _, item := range items {
		record := ReviewDataReview{
			ID:         firstReviewDataString(item, "id", "review_id"),
			ActorID:    firstReviewDataActorID(item),
			Actor:      firstReviewDataActor(item, "reviewer", "user", "author", "creator"),
			Status:     normalizeReviewDataStatus(firstReviewDataString(item, "status", "state", "review_status")),
			CommitID:   firstReviewDataString(item, "commit_id", "commit_sha", "sha"),
			Content:    firstReviewDataString(item, "content", "body", "note", "notes"),
			CreatedAt:  formatReviewDataTime(firstReviewDataTime(item, "created_at", "createdAt")),
			UpdatedAt:  formatReviewDataTime(firstReviewDataTime(item, "updated_at", "updatedAt")),
			SourceType: "formal_review",
		}
		record.Freshness = reviewDataFreshness(record.CommitID, headSHA)
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool { return reviewDataReviewSortKey(out[i]) < reviewDataReviewSortKey(out[j]) })
	return out
}

func normalizeReviewDataThreads(items []map[string]interface{}, headSHA string) []ReviewDataThread {
	out := make([]ReviewDataThread, 0, len(items))
	known := map[string]bool{}
	for _, item := range items {
		record := ReviewDataThread{
			ID:          firstReviewDataString(item, "id", "journal_id", "comment_id"),
			ReviewID:    reviewDataReviewID(item),
			ParentID:    firstReviewDataString(item, "parent_id"),
			AuthorID:    firstReviewDataActorID(item),
			Author:      firstReviewDataActor(item, "author", "user", "creator"),
			Content:     firstReviewDataString(item, "note", "content", "body"),
			Type:        strings.ToLower(strings.TrimSpace(firstReviewDataString(item, "type", "comment_type"))),
			State:       normalizeReviewDataThreadState(firstReviewDataString(item, "state", "status")),
			NeedRespond: firstReviewDataBoolPointer(item, "need_respond", "needRespond"),
			Path:        firstReviewDataString(item, "path", "file_path", "filename"),
			LineCode:    firstReviewDataString(item, "line_code", "lineCode"),
			CommitID:    firstReviewDataString(item, "commit_id", "commit_sha", "sha"),
			CreatedAt:   formatReviewDataTime(firstReviewDataTime(item, "created_at", "createdAt")),
			UpdatedAt:   formatReviewDataTime(firstReviewDataTime(item, "updated_at", "updatedAt")),
		}
		record.Freshness = reviewDataFreshness(record.CommitID, headSHA)
		if record.ID != "" {
			known[record.ID] = true
		}
		out = append(out, record)
	}
	for index := range out {
		out[index].UnknownParent = out[index].ParentID != "" && !known[out[index].ParentID]
	}
	sort.SliceStable(out, func(i, j int) bool { return reviewDataThreadSortKey(out[i]) < reviewDataThreadSortKey(out[j]) })
	return out
}

func summarizeReviewDataReviewers(reviews []ReviewDataReview) []ReviewDataReviewerSummary {
	grouped := map[string][]ReviewDataReview{}
	for _, review := range reviews {
		key := reviewDataReviewerKey(review)
		grouped[key] = append(grouped[key], review)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ReviewDataReviewerSummary, 0, len(keys))
	for _, key := range keys {
		records := grouped[key]
		summary := ReviewDataReviewerSummary{ReviewerKey: key, CurrentDecision: "none"}
		if len(records) > 0 {
			summary.ActorID, summary.Actor = records[0].ActorID, records[0].Actor
		}
		current := make([]ReviewDataReview, 0, len(records))
		for _, review := range records {
			switch review.Freshness {
			case "current":
				summary.CurrentCount++
				current = append(current, review)
			case "outdated":
				summary.OutdatedCount++
			default:
				summary.UnknownCount++
			}
		}
		if len(current) > 0 {
			latest, known, decision := latestReviewDataDecision(current)
			summary.CurrentDecision, summary.DecisionOrderKnown = decision, known
			if latest != nil {
				copy := *latest
				summary.LatestEffectiveReview = &copy
				summary.ActorID = firstReviewDataValue(copy.ActorID, summary.ActorID)
				summary.Actor = firstReviewDataValue(copy.Actor, summary.Actor)
			}
		}
		out = append(out, summary)
	}
	return out
}

func latestReviewDataDecision(reviews []ReviewDataReview) (*ReviewDataReview, bool, string) {
	if len(reviews) == 1 {
		copy := reviews[0]
		return &copy, true, copy.Status
	}
	latest := reviews[0]
	known := true
	statuses := map[string]bool{latest.Status: true}
	for _, candidate := range reviews[1:] {
		statuses[candidate.Status] = true
		comparison, comparable := compareReviewDataRecency(candidate, latest)
		if !comparable {
			known = false
			if reviewDataReviewSortKey(candidate) > reviewDataReviewSortKey(latest) {
				latest = candidate
			}
		} else if comparison > 0 {
			latest = candidate
		}
	}
	if !known {
		if len(statuses) == 1 {
			return nil, false, latest.Status
		}
		return nil, false, "unknown"
	}
	copy := latest
	return &copy, true, copy.Status
}

func compareReviewDataRecency(left, right ReviewDataReview) (int, bool) {
	leftTime := firstReviewDataParsedTime(left.UpdatedAt, left.CreatedAt)
	rightTime := firstReviewDataParsedTime(right.UpdatedAt, right.CreatedAt)
	if !leftTime.IsZero() || !rightTime.IsZero() {
		if !leftTime.IsZero() && !rightTime.IsZero() && !leftTime.Equal(rightTime) {
			if leftTime.After(rightTime) {
				return 1, true
			}
			return -1, true
		}
		return 0, false
	}
	leftID, leftOK := reviewDataNumericID(left.ID)
	rightID, rightOK := reviewDataNumericID(right.ID)
	if leftOK && rightOK && leftID != rightID {
		if leftID > rightID {
			return 1, true
		}
		return -1, true
	}
	return 0, false
}

func summarizeReviewData(reviews []ReviewDataReview, reviewers []ReviewDataReviewerSummary, threads []ReviewDataThread) ReviewDataSummary {
	summary := ReviewDataSummary{ReviewStatus: "none", ReviewFreshness: "unknown", Decision: "pending", TotalReviews: len(reviews), TotalThreads: len(threads)}
	for _, review := range reviews {
		switch review.Freshness {
		case "current":
			summary.CurrentReviews++
		case "outdated":
			summary.OutdatedReviews++
		default:
			summary.UnknownReviews++
		}
	}
	for _, thread := range threads {
		if thread.State == "opened" && thread.Freshness == "current" {
			summary.OpenThreads++
		}
	}
	if summary.CurrentReviews > 0 {
		summary.ReviewFreshness = "current"
	} else if summary.OutdatedReviews > 0 {
		summary.ReviewFreshness = "outdated"
	}
	hasApproved, hasCommon, hasRejected, hasUnknown := false, false, false, false
	for _, reviewer := range reviewers {
		if reviewer.CurrentCount == 0 {
			continue
		}
		switch reviewer.CurrentDecision {
		case "approved":
			hasApproved = true
		case "common":
			hasCommon = true
		case "rejected":
			hasRejected = true
		default:
			hasUnknown = true
		}
	}
	switch {
	case hasRejected:
		summary.ReviewStatus, summary.Decision = "rejected", "blocked"
	case hasUnknown:
		summary.ReviewStatus, summary.Decision = "unknown", "pending"
	case hasApproved:
		summary.ReviewStatus, summary.Decision = "approved", "approved"
	case hasCommon:
		summary.ReviewStatus, summary.Decision = "common", "commented"
	case len(reviews) > 0:
		summary.ReviewStatus = "unknown"
	}
	return summary
}

func reviewDataFreshness(commitID, headSHA string) string {
	left := strings.ToLower(strings.TrimSpace(commitID))
	right := strings.ToLower(strings.TrimSpace(headSHA))
	if left == "" || right == "" {
		return "unknown"
	}
	if left == right || (len(left) >= 7 && len(right) >= 7 && (strings.HasPrefix(left, right) || strings.HasPrefix(right, left))) {
		return "current"
	}
	return "outdated"
}

func normalizeReviewDataPRState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open", "opened", "0":
		return "open"
	case "merged", "merge", "1":
		return "merged"
	case "closed", "close", "2":
		return "closed"
	default:
		return "unknown"
	}
}

func normalizeReviewDataStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "common", "comment", "commented":
		return "common"
	case "approved", "approve":
		return "approved"
	case "rejected", "reject", "changes_requested", "request_changes":
		return "rejected"
	default:
		return "unknown"
	}
}

func normalizeReviewDataThreadState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open", "opened":
		return "opened"
	case "resolve", "resolved":
		return "resolved"
	case "disable", "disabled":
		return "disabled"
	default:
		return "unknown"
	}
}

func firstReviewDataString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := reviewDataString(item[key]); value != "" {
			return value
		}
	}
	return ""
}

func reviewDataString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case map[string]interface{}:
		return firstReviewDataString(typed, "login", "name", "username", "full_name", "display_name", "id", "sha", "ref")
	}
	return ""
}

func firstReviewDataInt(item map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		switch value := item[key].(type) {
		case int:
			return value
		case float64:
			return int(value)
		case json.Number:
			parsed, _ := strconv.Atoi(value.String())
			return parsed
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func reviewDataBool(item map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		switch value := item[key].(type) {
		case bool:
			return value, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err == nil {
				return parsed, true
			}
		case float64:
			return value != 0, true
		}
	}
	return false, false
}

func firstReviewDataBoolPointer(item map[string]interface{}, keys ...string) *bool {
	if value, ok := reviewDataBool(item, keys...); ok {
		return &value
	}
	return nil
}

func firstReviewDataActor(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if nested, ok := item[key].(map[string]interface{}); ok {
			if value := firstReviewDataString(nested, "login", "username", "name", "full_name", "display_name"); value != "" {
				return value
			}
		}
		if value := reviewDataString(item[key]); value != "" {
			return value
		}
	}
	return firstReviewDataString(item, "author_name", "reviewer_name", "username", "login")
}

func firstReviewDataActorID(item map[string]interface{}) string {
	if value := firstReviewDataString(item, "reviewer_id", "user_id", "author_id", "creator_id"); value != "" {
		return value
	}
	for _, key := range []string{"reviewer", "user", "author", "creator"} {
		if nested, ok := item[key].(map[string]interface{}); ok {
			if value := firstReviewDataString(nested, "id", "user_id", "uid"); value != "" {
				return value
			}
		}
	}
	return ""
}

func firstReviewDataBranch(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if nested, ok := item[key].(map[string]interface{}); ok {
			if value := firstReviewDataString(nested, "ref", "branch", "name"); value != "" {
				return value
			}
		}
		if value := reviewDataString(item[key]); value != "" {
			return value
		}
	}
	return ""
}

func reviewDataReviewID(item map[string]interface{}) string {
	if value := firstReviewDataString(item, "review_id"); value != "" {
		return value
	}
	if nested, ok := item["review"].(map[string]interface{}); ok {
		return firstReviewDataString(nested, "id", "review_id")
	}
	return ""
}

func firstReviewDataTime(item map[string]interface{}, keys ...string) time.Time {
	for _, key := range keys {
		if parsed := parseReviewDataTime(reviewDataString(item[key])); !parsed.IsZero() {
			return parsed
		}
	}
	return time.Time{}
}

func parseReviewDataTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed
		}
	}
	china := time.FixedZone("Asia/Shanghai", 8*60*60)
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, trimmed, china); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func formatReviewDataTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func firstReviewDataParsedTime(values ...string) time.Time {
	for _, value := range values {
		if parsed := parseReviewDataTime(value); !parsed.IsZero() {
			return parsed
		}
	}
	return time.Time{}
}

func reviewDataNumericID(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed, err == nil
}

func reviewDataReviewerKey(review ReviewDataReview) string {
	if review.ActorID != "" {
		return "id:" + strings.ToLower(review.ActorID)
	}
	if review.Actor != "" {
		return "actor:" + strings.ToLower(review.Actor)
	}
	return "unknown-review:" + firstReviewDataValue(review.ID, reviewDataReviewSortKey(review))
}

func reviewDataReviewSortKey(review ReviewDataReview) string {
	return strings.Join([]string{review.ID, review.ActorID, review.Actor, review.CreatedAt, review.UpdatedAt, review.CommitID, review.Status, review.Content}, "\x00")
}

func reviewDataThreadSortKey(thread ReviewDataThread) string {
	return strings.Join([]string{thread.ID, thread.ParentID, thread.AuthorID, thread.CreatedAt, thread.UpdatedAt, thread.Path, thread.LineCode, thread.Content}, "\x00")
}

func firstReviewDataValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sortedReviewDataStrings(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = true
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
