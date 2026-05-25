package health

import (
	"database/sql"
	_ "embed"
	"fmt"

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
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE user_name = ?", username).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := db.Exec("INSERT INTO users (user_name) VALUES (?)", username)
	if err != nil {
		return 0, fmt.Errorf("insert user %q: %w", username, err)
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
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
	var id int
	err := db.QueryRow("SELECT id FROM tags WHERE repo_id = ? AND name = ?", repoID, tagName).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := db.Exec("INSERT INTO tags (repo_id, name) VALUES (?, ?)", repoID, tagName)
	if err != nil {
		return 0, fmt.Errorf("insert tag %q for repo %d: %w", tagName, repoID, err)
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
}

func savePullTags(db *sql.DB, pullID int, tagIDs []int) {
	if len(tagIDs) == 0 {
		return
	}
	for _, tagID := range tagIDs {
		db.Exec("INSERT OR IGNORE INTO pull_tags (pull_id, tag_id) VALUES (?, ?)", pullID, tagID)
	}
}

func saveIssueTags(db *sql.DB, issueID int, tagIDs []int) {
	if len(tagIDs) == 0 {
		return
	}
	for _, tagID := range tagIDs {
		db.Exec("INSERT OR IGNORE INTO issue_tags (issue_id, tag_id) VALUES (?, ?)", issueID, tagID)
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

func savePull(db *sql.DB, repoID int, pr map[string]interface{}) {
	// id: pull_request_id preferred, fallback to id
	var prID float64
	if v, ok := pr["pull_request_id"].(float64); ok && v > 0 {
		prID = v
	} else if v, ok := pr["id"].(float64); ok {
		prID = v
	} else {
		return
	}

	prNumber, _ := pr["pull_request_number"].(float64)
	if prNumber == 0 {
		return
	}

	author, _ := pr["author_login"].(string)
	createrID, _ := getOrCreateUser(db, author)

	statusCode := 0
	if v, ok := pr["pull_request_status"].(float64); ok {
		statusCode = int(v)
	}
	statusMap := map[int]string{0: "open", 1: "merged", 2: "closed"}
	status := statusMap[statusCode]
	if status == "" {
		status = "open"
	}

	createTime, _ := pr["pr_full_time"].(string)

	var processorID *int
	if assignee, _ := pr["assign_user_login"].(string); assignee != "" {
		pid, _ := getOrCreateUser(db, assignee)
		processorID = &pid
	}

	db.Exec(`INSERT OR REPLACE INTO pulls (id, repo_id, number, creater_id, status, processor_id, create_time, close_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		int(prID), repoID, int(prNumber), createrID, status, processorID, createTime, nil)

	// Save tags
	tagNames := extractTagNames(pr, "issue_tags")
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
	if statusName == "关闭" {
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

	db.Exec(`INSERT OR REPLACE INTO issues (id, repo_id, number, creater_id, processor_id, create_time, close_time, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		int(issueID), repoID, issueNumber, createrID, processorID, createTime, closeTime, status)

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
