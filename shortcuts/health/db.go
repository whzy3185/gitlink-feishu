package health

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// extractLogin extracts a login string from either a nested object {"login": "..."} or a flat string field.
func extractLogin(data map[string]interface{}, objKey, flatKey string) string {
	if obj, ok := data[objKey].(map[string]interface{}); ok {
		if login, _ := obj["login"].(string); login != "" {
			return login
		}
	}
	if login, _ := data[flatKey].(string); login != "" {
		return login
	}
	return ""
}

// extractStatusName extracts the status name from either a nested object {"name": "关闭"} or an issue_status string.
func extractStatusName(data map[string]interface{}) string {
	if obj, ok := data["status"].(map[string]interface{}); ok {
		if name, _ := obj["name"].(string); name != "" {
			return name
		}
	}
	if name, _ := data["issue_status"].(string); name != "" {
		return name
	}
	return ""
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return db, nil
}

func getOrCreateUser(db *sql.DB, username string) (int, error) {
	if username == "" {
		return 0, nil
	}
	if _, err := db.Exec("INSERT OR IGNORE INTO users (user_name) VALUES (?)", username); err != nil {
		return 0, fmt.Errorf("insert user %q: %w", username, err)
	}
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE user_name = ?", username).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get user %q: %w", username, err)
	}
	return id, nil
}

func getOrCreateRepo(db *sql.DB, repoName, owner string) (int, error) {
	ownerID, err := getOrCreateUser(db, owner)
	if err != nil {
		return 0, err
	}
	var id int
	err = db.QueryRow("SELECT id FROM repos WHERE repo_name = ? AND owner_id = ?", repoName, ownerID).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := db.Exec("INSERT INTO repos (repo_name, owner_id) VALUES (?, ?)", repoName, ownerID)
	if err != nil {
		return 0, fmt.Errorf("insert repo %s/%s: %w", owner, repoName, err)
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
}

func getOrCreateTag(db *sql.DB, repoID int, tagName string) (int, error) {
	if _, err := db.Exec("INSERT OR IGNORE INTO tags (repo_id, name) VALUES (?, ?)", repoID, tagName); err != nil {
		return 0, fmt.Errorf("insert tag %q for repo %d: %w", tagName, repoID, err)
	}
	var id int
	err := db.QueryRow("SELECT id FROM tags WHERE repo_id = ? AND name = ?", repoID, tagName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get tag %q for repo %d: %w", tagName, repoID, err)
	}
	return id, nil
}

func savePullTags(db *sql.DB, pullID int, tagIDs []int) {
	if len(tagIDs) == 0 {
		return
	}
	for _, tagID := range tagIDs {
		if _, err := db.Exec("INSERT OR IGNORE INTO pull_tags (pull_id, tag_id) VALUES (?, ?)", pullID, tagID); err != nil {
			fmt.Fprintf(os.Stderr, "  DB error: save pull_tags (pull=%d tag=%d): %v\n", pullID, tagID, err)
		}
	}
}

func saveIssueTags(db *sql.DB, issueID int, tagIDs []int) {
	if len(tagIDs) == 0 {
		return
	}
	for _, tagID := range tagIDs {
		if _, err := db.Exec("INSERT OR IGNORE INTO issue_tags (issue_id, tag_id) VALUES (?, ?)", issueID, tagID); err != nil {
			fmt.Fprintf(os.Stderr, "  DB error: save issue_tags (issue=%d tag=%d): %v\n", issueID, tagID, err)
		}
	}
}

func extractTagNames(data map[string]interface{}, key string) []string {
	rawTags, ok := data[key].([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, item := range rawTags {
		tag, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := tag["name"].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func nestedMap(data map[string]interface{}, key string) map[string]interface{} {
	obj, _ := data[key].(map[string]interface{})
	return obj
}

func extractPullNumber(pr map[string]interface{}) int {
	for _, key := range []string{"pull_request_number", "index"} {
		if v, ok := pr[key].(float64); ok && v > 0 {
			return int(v)
		}
		if v, ok := pr[key].(int); ok && v > 0 {
			return v
		}
	}
	return 0
}

func extractPullAuthorLogin(pr map[string]interface{}) string {
	if login, _ := pr["author_login"].(string); login != "" {
		return login
	}
	if login := extractLogin(pr, "author", ""); login != "" {
		return login
	}
	if issue := nestedMap(pr, "issue"); issue != nil {
		return extractLogin(issue, "author", "")
	}
	return ""
}

func extractPullAssigneeLogin(pr map[string]interface{}) string {
	if login, _ := pr["assign_user_login"].(string); login != "" {
		return login
	}
	if issue := nestedMap(pr, "issue"); issue != nil {
		if login := extractLogin(issue, "assign_user", "assign_user_login"); login != "" {
			return login
		}
	}
	return ""
}

func extractPullStatus(pr map[string]interface{}) string {
	statusMap := map[int]string{0: "open", 1: "merged", 2: "closed"}
	if v, ok := pr["pull_request_status"].(float64); ok {
		return statusMap[int(v)]
	}
	if v, ok := pr["pull_request_status"].(int); ok {
		return statusMap[v]
	}
	for _, key := range []string{"pull_request_staus", "status", "state"} {
		if status, _ := pr[key].(string); status != "" {
			switch status {
			case "open", "opened":
				return "open"
			case "merged":
				return "merged"
			case "closed", "close":
				return "closed"
			}
		}
	}
	return "open"
}

func extractPullCreateTime(pr map[string]interface{}) string {
	for _, key := range []string{"pr_full_time", "created_at", "create_time"} {
		if v, _ := pr[key].(string); v != "" {
			return v
		}
	}
	if issue := nestedMap(pr, "issue"); issue != nil {
		for _, key := range []string{"created_at", "create_time"} {
			if v, _ := issue[key].(string); v != "" {
				return v
			}
		}
	}
	return ""
}

func extractPullTagNames(pr map[string]interface{}) []string {
	if names := extractTagNames(pr, "issue_tags"); len(names) > 0 {
		return names
	}
	if issue := nestedMap(pr, "issue"); issue != nil {
		return extractTagNames(issue, "issue_tags")
	}
	return nil
}

// mergeTimeFromList tries to extract merged_at directly from the list API response
// (preferred, avoids an extra API call per PR).
func mergeTimeFromList(pr map[string]interface{}) string {
	for _, key := range []string{"merged_at", "mergedAt", "pr_merge_time", "merge_time"} {
		if v, _ := pr[key].(string); v != "" {
			return v
		}
		// Some APIs return numeric timestamps
		if v, ok := pr[key].(float64); ok && v > 0 {
			return fmt.Sprintf("%.0f", v)
		}
	}
	return ""
}

func savePull(db *sql.DB, repoID int, pr map[string]interface{}, mergedAt string) {
	// id: pull_request_id preferred, fallback to id
	var prID float64
	if v, ok := pr["pull_request_id"].(float64); ok && v > 0 {
		prID = v
	} else if v, ok := pr["id"].(float64); ok {
		prID = v
	} else {
		return
	}

	prNumber := extractPullNumber(pr)
	if prNumber == 0 {
		return
	}

	createrID, _ := getOrCreateUser(db, extractPullAuthorLogin(pr))

	status := extractPullStatus(pr)
	if status == "" {
		status = "open"
	}

	createTime := extractPullCreateTime(pr)

	var processorID *int
	if assignee := extractPullAssigneeLogin(pr); assignee != "" {
		pid, _ := getOrCreateUser(db, assignee)
		processorID = &pid
	}

	// Priority: explicit mergedAt arg > list response field > nil
	var mergedAtVal interface{} = nil
	if mergedAt != "" {
		mergedAtVal = mergedAt
	} else if ma := mergeTimeFromList(pr); ma != "" {
		mergedAtVal = ma
	}

	if _, err := db.Exec(`INSERT OR REPLACE INTO pulls (id, repo_id, number, creater_id, status, processor_id, create_time, merged_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		int(prID), repoID, prNumber, createrID, status, processorID, createTime, mergedAtVal); err != nil {
		fmt.Fprintf(os.Stderr, "  DB error: save pull %d: %v\n", int(prID), err)
	}

	// Save tags
	tagNames := extractPullTagNames(pr)
	var tagIDs []int
	for _, name := range tagNames {
		if tid, err := getOrCreateTag(db, repoID, name); err == nil {
			tagIDs = append(tagIDs, tid)
		}
	}
	savePullTags(db, int(prID), tagIDs)
}

func saveIssue(db *sql.DB, repoID int, issue map[string]interface{}, issueNumber int, listUpdatedAt string) {
	issueID, ok := issue["id"].(float64)
	if !ok || issueID == 0 {
		return
	}

	createrID, _ := getOrCreateUser(db, extractLogin(issue, "author", "author_login"))

	var processorID *int
	if login := extractLogin(issue, "assign_user", "assign_user_login"); login != "" {
		pid, _ := getOrCreateUser(db, login)
		processorID = &pid
	}

	createTime, _ := issue["created_at"].(string)

	statusName := extractStatusName(issue)
	var status string
	var closeTime interface{}
	if statusName == "关闭" || statusName == "Closed" {
		status = "close"
		if v, _ := issue["closed_on"].(string); v != "" {
			closeTime = v
		} else if v, _ := issue["updated_at"].(string); v != "" {
			closeTime = v
		} else if listUpdatedAt != "" {
			closeTime = listUpdatedAt
		}
	} else {
		status = "open"
	}

	if _, err := db.Exec(`INSERT OR REPLACE INTO issues (id, repo_id, number, creater_id, processor_id, create_time, close_time, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		int(issueID), repoID, issueNumber, createrID, processorID, createTime, closeTime, status); err != nil {
		fmt.Fprintf(os.Stderr, "  DB error: save issue %d: %v\n", int(issueID), err)
	}

	// Save tags
	tagNames := extractTagNames(issue, "tags")
	var tagIDs []int
	for _, name := range tagNames {
		if tid, err := getOrCreateTag(db, repoID, name); err == nil {
			tagIDs = append(tagIDs, tid)
		}
	}
	saveIssueTags(db, int(issueID), tagIDs)
}
