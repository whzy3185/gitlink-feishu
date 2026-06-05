package health

import (
	"path/filepath"
	"testing"
)

func TestSavePullAcceptsV1ListShape(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	repoID, err := getOrCreateRepo(db, "repo", "owner")
	if err != nil {
		t.Fatalf("getOrCreateRepo: %v", err)
	}

	savePull(db, repoID, map[string]interface{}{
		"id":     float64(15414),
		"index":  float64(109),
		"status": "merged",
		"issue": map[string]interface{}{
			"author": map[string]interface{}{"login": "alice"},
			"issue_tags": []interface{}{
				map[string]interface{}{"name": "docs"},
			},
		},
	}, "2026-06-05T12:00:00+08:00")

	var number int
	var status string
	var author string
	var mergedAt string
	if err := db.QueryRow(`
		SELECT pulls.number, pulls.status, users.user_name, pulls.merged_at
		FROM pulls JOIN users ON users.id = pulls.creater_id
		WHERE pulls.id = ?`, 15414).Scan(&number, &status, &author, &mergedAt); err != nil {
		t.Fatalf("query saved pull: %v", err)
	}
	if number != 109 {
		t.Fatalf("number=%d, want 109", number)
	}
	if status != "merged" {
		t.Fatalf("status=%q, want merged", status)
	}
	if author != "alice" {
		t.Fatalf("author=%q, want alice", author)
	}
	if mergedAt == "" {
		t.Fatal("merged_at was not saved")
	}

	var tagCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pull_tags WHERE pull_id = ?`, 15414).Scan(&tagCount); err != nil {
		t.Fatalf("query tags: %v", err)
	}
	if tagCount != 1 {
		t.Fatalf("tagCount=%d, want 1", tagCount)
	}
}
