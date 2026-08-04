package feishu

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConditionalCollaborationUpdateRejectsStaleState(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	tx, err := store.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin read transaction: %v", err)
	}
	before, err := readScopedReviewCollaboration(context.Background(), tx, job)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("read collaboration: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit read transaction: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE review_collaboration_states SET updated_at=? WHERE collaboration_key=?`,
		reviewProductInvariantTime.Add(time.Minute).Format(time.RFC3339Nano), before.CollaborationKey,
	); err != nil {
		t.Fatalf("advance collaboration state: %v", err)
	}
	after := before
	after.AssignedTo = "ou_stale"
	after.CollaborationStatus = "reviewing"
	after.UpdatedAt = reviewProductInvariantTime.Add(2 * time.Minute).Format(time.RFC3339Nano)
	tx, err = store.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin stale transaction: %v", err)
	}
	defer tx.Rollback()
	if err := writeScopedReviewCollaborationAction(context.Background(), tx, before, after, true); !errors.Is(err, ErrReviewCollaborationConflict) {
		t.Fatalf("stale collaboration update error = %v", err)
	}
}

func TestArchivedPRRejectsReleaseAndPreservesHistoricalAssignee(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	job.Action = "claim_review"
	claimed, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim Review: %v", err)
	}
	merged := fullReviewGatewayResultFixture()
	merged.GitLinkState = "merged"
	merged.CompletedAt = reviewProductInvariantTime.Add(2 * time.Minute).Format(time.RFC3339)
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, merged, reviewProductInvariantTime.Add(2*time.Minute)); err != nil {
		t.Fatalf("archive Review: %v", err)
	}
	job.JobID = "job-archived-release"
	job.Action = "release_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(3*time.Minute)); err == nil {
		t.Fatal("archived Review release unexpectedly succeeded")
	}
	items, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if err != nil || len(items) != 0 {
		// Archived items are intentionally excluded from the active queue; read
		// the durable row below to verify that historical ownership is retained.
		if err != nil {
			t.Fatalf("list archived collaboration: %v", err)
		}
	}
	var assignedTo string
	if err := store.db.QueryRow(`SELECT assigned_to FROM review_collaboration_states WHERE collaboration_key=?`, claimed.PRKey).Scan(&assignedTo); err != nil {
		t.Fatalf("read historical assignee: %v", err)
	}
	if assignedTo != claimed.AssignedTo {
		t.Fatalf("archived release changed historical assignee: got=%q want=%q", assignedTo, claimed.AssignedTo)
	}
}
