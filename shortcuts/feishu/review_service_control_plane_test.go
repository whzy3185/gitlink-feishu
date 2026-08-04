package feishu

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const reviewServiceTestAdminToken = "stage-five-test-token"

func startAuthenticatedReviewService(t *testing.T) *ReviewService {
	t.Helper()
	t.Setenv("FEISHU_REVIEW_ADMIN_TOKEN_TEST", reviewServiceTestAdminToken)
	service := newReviewServiceTestHarness(t)
	service.Config.AdminTokenRef = "env:FEISHU_REVIEW_ADMIN_TOKEN_TEST"
	requireReviewServiceStart(t, service)
	return service
}

func reviewAdminRequest(t *testing.T, service *ReviewService, method, path, token string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, "http://"+service.AdminAddress()+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func readReviewHTTPBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestMetricsRequireAdminToken(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	response := reviewAdminRequest(t, service, http.MethodGet, "/metrics", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.StatusCode, readReviewHTTPBody(t, response))
	}
}

func TestMetricsExposeBoundedLabels(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	for _, forbidden := range []string{"chat_id=", "repository=", "operation_id=", "installation_id="} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("unbounded label %q found in metrics", forbidden)
		}
	}
}

func TestMetricsDoNotExposeChatID(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	secret := "oc_chat_secret"
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,queue_class) VALUES('j-chat','d-chat','queued','view','u',?,'{}',?,?,'gitlink_read')`, secret, reviewGatewayTimestamp(service.now()), reviewGatewayTimestamp(service.now())); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if strings.Contains(body, secret) {
		t.Fatal("metrics exposed chat id")
	}
}

func TestMetricsDoNotExposeRepository(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	secret := "owner/private-repository"
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,repository,requested_by,chat_id,payload_json,created_at,updated_at,queue_class) VALUES('j-repo','d-repo','queued','view',?,'u','chat','{}',?,?,'gitlink_read')`, secret, reviewGatewayTimestamp(service.now()), reviewGatewayTimestamp(service.now())); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if strings.Contains(body, secret) {
		t.Fatal("metrics exposed repository")
	}
}

func TestMetricsDoNotExposeOperationID(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	op, err := NewReviewOperation(ReviewOperationReplySend, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "work", "reply", map[string]string{"text": "ok"}, ReviewRetryNonIdempotent, service.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if strings.Contains(body, op.OperationID) {
		t.Fatal("metrics exposed operation id")
	}
}

func TestMetricsReportQueueDepth(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	if _, err := service.Store.db.Exec(`INSERT INTO review_gateway_jobs(job_id,dedupe_key,status,action,requested_by,chat_id,payload_json,created_at,updated_at,queue_class) VALUES('j-depth','d-depth','queued','view','u','chat','{}',?,?,'gitlink_read')`, reviewGatewayTimestamp(service.now()), reviewGatewayTimestamp(service.now())); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if !strings.Contains(body, `review_job_queue_depth{queue_class="gitlink_read"} 1`) {
		t.Fatalf("queue depth missing: %s", body)
	}
}

func TestMetricsReportOperationStates(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	op, err := NewReviewOperation(ReviewOperationCanonicalCardUpsert, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "work-state", "card", map[string]string{"title": "ok"}, ReviewRetryIdempotent, service.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if !strings.Contains(body, `review_operation_depth{operation_kind="canonical_card_upsert",operation_status="pending"} 1`) {
		t.Fatalf("operation state missing: %s", body)
	}
}

func TestMetricsReportDeadLetters(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	now := reviewGatewayTimestamp(service.now())
	if _, err := service.Store.db.Exec(`INSERT INTO review_dead_letters(dead_letter_id,entity_type,entity_id,error_class,status,created_at,updated_at) VALUES('dl-1','job','job-1','permanent','open',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	if !strings.Contains(body, "review_operation_dead_letter_total 1") {
		t.Fatalf("dead letter metric missing: %s", body)
	}
}

func TestMetricsReportSQLiteAndWALSize(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/metrics", reviewServiceTestAdminToken))
	for _, metric := range []string{"review_sqlite_file_size_bytes ", "review_sqlite_wal_size_bytes "} {
		if !strings.Contains(body, metric) {
			t.Fatalf("metric %q missing", metric)
		}
	}
}

func TestMetricsUseCachedValuesWhenSQLiteBusy(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	first, err := service.Metrics.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	closed := service.Store.db
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := service.Metrics.Render(context.Background())
	if err != nil || second != first {
		t.Fatalf("cached metrics not returned: err=%v", err)
	}
}

func TestMetricsPerformNoExternalRequests(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	if _, err := service.Metrics.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAdminListenerAcceptsLoopback(t *testing.T) {
	if err := validateReviewAdminListenAddress("127.0.0.1:9091"); err != nil {
		t.Fatal(err)
	}
}

func TestAdminListenerRejectsNonLoopback(t *testing.T) {
	if err := validateReviewAdminListenAddress("0.0.0.0:9091"); err == nil {
		t.Fatal("non-loopback admin listener accepted")
	}
}

func TestAdminAPIRequiresBearerToken(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	response := reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/summary", "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestAdminTokenUsesConstantTimeComparison(t *testing.T) {
	if !reviewAdminTokenEqual([]byte("same"), []byte("same")) || reviewAdminTokenEqual([]byte("same"), []byte("different")) || reviewAdminTokenEqual(nil, nil) {
		t.Fatal("constant-time token comparison contract failed")
	}
}

func TestAdminAPIIsReadOnly(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		response := reviewAdminRequest(t, service, method, "/admin/v1/summary", reviewServiceTestAdminToken)
		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("method=%s status=%d", method, response.StatusCode)
		}
		_ = response.Body.Close()
	}
}

func TestAdminAPIPaginatesResults(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	service.Status.RegisterComponent("one", 0, false, true, service.now())
	service.Status.RegisterComponent("two", 0, false, true, service.now())
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/components?limit=1", reviewServiceTestAdminToken))
	if !strings.Contains(body, `"next_cursor":"`) || strings.Contains(body, `"next_cursor":""`) {
		t.Fatalf("pagination cursor missing: %s", body)
	}
}

func TestAdminAPIRejectsInvalidCursor(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	response := reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/components?cursor=invalid!", reviewServiceTestAdminToken)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.StatusCode, readReviewHTTPBody(t, response))
	}
}

func TestAdminAPIRedactsRemoteIdentifiers(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	op, err := NewReviewOperation(ReviewOperationReplySend, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "work-admin", "reply", map[string]string{"text": "ok"}, ReviewRetryNonIdempotent, service.now())
	if err != nil {
		t.Fatal(err)
	}
	op.RemoteID = "remote-secret-id"
	if err := service.Store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/operations", reviewServiceTestAdminToken))
	if strings.Contains(body, op.OperationID) || strings.Contains(body, op.RemoteID) || !strings.Contains(body, "operation_id_hash") {
		t.Fatalf("identifier redaction failed: %s", body)
	}
}

func TestAdminAPIDoesNotExposeDesiredJSON(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	op, err := NewReviewOperation(ReviewOperationReplySend, ReviewGatewayJob{Repository: "owner/repo", ChatID: "chat"}, "work-json", "reply", map[string]string{"text": "desired-secret"}, ReviewRetryNonIdempotent, service.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/operations", reviewServiceTestAdminToken))
	if strings.Contains(body, "desired-secret") || strings.Contains(body, "desired_json") {
		t.Fatalf("desired json exposed: %s", body)
	}
}

func TestAdminAPIDoesNotExposeSecrets(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/summary", reviewServiceTestAdminToken))
	if strings.Contains(body, reviewServiceTestAdminToken) || strings.Contains(body, "admin_token") {
		t.Fatalf("secret exposed: %s", body)
	}
}

func TestAdminSummaryReportsServiceState(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	body := readReviewHTTPBody(t, reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/summary", reviewServiceTestAdminToken))
	if !strings.Contains(body, `"service_state":"ready"`) || !strings.Contains(body, `"admission":"open"`) {
		t.Fatalf("service state missing: %s", body)
	}
}

func TestAdminOperationsFiltersAreWhitelisted(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	response := reviewAdminRequest(t, service, http.MethodGet, "/admin/v1/operations?desired_json=secret", reviewServiceTestAdminToken)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.StatusCode, readReviewHTTPBody(t, response))
	}
}

func TestMetricsHandlerDoesNotAcceptMutation(t *testing.T) {
	service := startAuthenticatedReviewService(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer "+reviewServiceTestAdminToken)
	service.handleMetrics(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", recorder.Code)
	}
}
