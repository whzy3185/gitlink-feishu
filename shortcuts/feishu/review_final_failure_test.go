package feishu

import (
	"context"
	"testing"
	"time"
)

func TestFinalCardPatchTimeoutSchedulesSafeRetry(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	first, err := NewReviewOperation(
		ReviewOperationCanonicalCardUpsert,
		job,
		reviewChatPRPresentationKey(job),
		ReviewResourceCard,
		reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime), Card: Card{"value": "one"}},
		ReviewRetryNonIdempotent,
		reviewOperationTestTime,
	)
	if err != nil {
		t.Fatal(err)
	}
	executeReviewOperationForTest(t, store, handler, first)
	sender.err = context.DeadlineExceeded
	second, err := NewReviewOperation(
		ReviewOperationCanonicalCardUpsert,
		job,
		reviewChatPRPresentationKey(job),
		ReviewResourceCard,
		reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime.Add(time.Minute)), Card: Card{"value": "two"}},
		ReviewRetryNonIdempotent,
		reviewOperationTestTime.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	stored := executeReviewOperationForTest(t, store, handler, second)
	if stored.Status != ReviewOperationRetryScheduled || stored.RequiresReconciliation || sender.patches != 1 {
		t.Fatalf("status=%s reconcile=%t patches=%d", stored.Status, stored.RequiresReconciliation, sender.patches)
	}
}
