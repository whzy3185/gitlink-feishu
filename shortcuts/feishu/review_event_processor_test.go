package feishu

import (
	"context"
	"testing"
	"time"
)

func testReviewEventRoutesOnlyGroup(t *testing.T, group, eventName, action string) {
	t.Helper()
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{
		reviewSubscriptionBinding("chat-match", true, "admin-match"),
		reviewSubscriptionBinding("chat-other", true, "admin-other"),
	})
	if _, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-match", Repository: "owner/repo", ActorID: "admin-match",
		AddGroups: []string{group},
	}, reviewEventTestTime); err != nil {
		t.Fatal(err)
	}
	other := ReviewSubscriptionGroupPulls
	if group == other {
		other = ReviewSubscriptionGroupReviews
	}
	if _, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-other", Repository: "owner/repo", ActorID: "admin-other",
		AddGroups: []string{other},
	}, reviewEventTestTime); err != nil {
		t.Fatal(err)
	}
	record := insertReviewEventTest(t, store, eventName, action)
	queue := &recordingReviewEventQueue{}
	processor := &ReviewEventInboxProcessor{Store: store, Queue: queue, LeaseOwner: "route-worker", Now: func() time.Time { return reviewEventTestTime }}
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessOne = %v, %v", processed, err)
	}
	routes, err := store.ListReviewEventRoutes(context.Background(), record.Event.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || len(queue.jobs) != 1 || routes[0].ChatIDHash != reviewResourceIdentifierHash("chat-match") {
		t.Fatalf("routes=%+v jobs=%+v", routes, queue.jobs)
	}
}

func TestPullEventRoutesOnlyPullSubscribers(t *testing.T) {
	testReviewEventRoutesOnlyGroup(t, ReviewSubscriptionGroupPulls, "pull_request", "opened")
}

func TestMergeEventRoutesOnlyMergeSubscribers(t *testing.T) {
	testReviewEventRoutesOnlyGroup(t, ReviewSubscriptionGroupMerge, "pull_request", "merged")
}

func TestReviewEventRoutesOnlyReviewSubscribers(t *testing.T) {
	testReviewEventRoutesOnlyGroup(t, ReviewSubscriptionGroupReviews, "review", "submitted")
}

func TestThreadEventRoutesOnlyThreadSubscribers(t *testing.T) {
	testReviewEventRoutesOnlyGroup(t, ReviewSubscriptionGroupThreads, "review_thread", "resolved")
}

func TestCIEventRoutesOnlyCISubscribers(t *testing.T) {
	testReviewEventRoutesOnlyGroup(t, ReviewSubscriptionGroupCI, "check_run", "completed")
}

func TestRouteUsesCurrentSubscriptionRevision(t *testing.T) {
	store, _, processor, record := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	current, err := store.GetReviewChatSubscription(context.Background(), "installation-a", "chat-a", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupReviews}, ExpectedRevision: current.Revision,
	}, reviewEventTestTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = processor.ProcessOne(context.Background())
	routes, _ := store.ListReviewEventRoutes(context.Background(), record.Event.EventID)
	if len(routes) != 1 || routes[0].SubscriptionRevision != updated.Revision {
		t.Fatalf("route did not capture current subscription revision: routes=%+v subscription=%+v", routes, updated)
	}
}

func TestRouteCreatesReadOnlyRefreshJob(t *testing.T) {
	_, queue, processor, _ := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 {
		t.Fatalf("jobs=%d", len(queue.jobs))
	}
	job := queue.jobs[0]
	if job.Action != "refresh_review_context" || job.Mode != "preview" || job.RequestedBy != "gitlink-event:installation-a" {
		t.Fatalf("unexpected route job: %+v", job)
	}
}

func TestRouteNeverSetsMutatesGitLink(t *testing.T) {
	_, queue, processor, _ := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalAndNotice, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 || queue.jobs[0].MutatesGitLink {
		t.Fatalf("route job can mutate GitLink: %+v", queue.jobs)
	}
}

func TestDuplicateRouteCreatesNoSecondJob(t *testing.T) {
	store, queue, processor, record := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	_, _ = store.db.Exec(`UPDATE review_event_inbox SET status='normalized', processed_at='' WHERE event_id=?`, record.Event.EventID)
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 {
		t.Fatalf("duplicate route created %d jobs", len(queue.jobs))
	}
}

func TestCanonicalOnlyCreatesNoNotice(t *testing.T) {
	_, queue, processor, _ := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 || queue.jobs[0].NotifyChat || queue.jobs[0].NotificationMode != ReviewNotificationCanonicalOnly || !shouldDispatchReviewGatewayResult(queue.jobs[0]) {
		t.Fatalf("canonical-only job = %+v", queue.jobs)
	}
}

func TestCanonicalAndNoticeEnablesLightNotice(t *testing.T) {
	_, queue, processor, _ := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalAndNotice, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 || !queue.jobs[0].NotifyChat || queue.jobs[0].NotificationMode != ReviewNotificationCanonicalAndNotice {
		t.Fatalf("canonical-and-notice job = %+v", queue.jobs)
	}
}

func TestSilentRefreshDisablesNotice(t *testing.T) {
	_, queue, processor, _ := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationSilentRefresh, "pull_request", "opened")
	_, _ = processor.ProcessOne(context.Background())
	if len(queue.jobs) != 1 || queue.jobs[0].NotifyChat || shouldDispatchReviewGatewayResult(queue.jobs[0]) {
		t.Fatalf("silent-refresh job = %+v", queue.jobs)
	}
}
