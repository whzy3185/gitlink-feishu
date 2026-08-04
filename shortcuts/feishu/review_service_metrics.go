package feishu

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var reviewMetricAllowedLabels = map[string]bool{"component": true, "queue_class": true, "operation_kind": true, "operation_status": true, "error_class": true, "resource_type": true, "result": true}

type ReviewMetrics struct {
	Store  *SQLiteReviewGatewayStore
	Status *ReviewServiceStatusRegistry
	DBPath string
	Now    func() time.Time

	mu     sync.RWMutex
	cached string

	jobRetries       atomic.Uint64
	jobCoalesced     atomic.Uint64
	admissionRejects atomic.Uint64
	rateRejects      atomic.Uint64
	sqliteBusy       atomic.Uint64

	backupSuccess    atomic.Int64
	backupFailure    atomic.Int64
	retentionSuccess atomic.Int64
	walSuccess       atomic.Int64
}

func NewReviewMetrics(store *SQLiteReviewGatewayStore, status *ReviewServiceStatusRegistry, path string, now func() time.Time) *ReviewMetrics {
	if now == nil {
		now = time.Now
	}
	return &ReviewMetrics{Store: store, Status: status, DBPath: path, Now: now}
}

func (m *ReviewMetrics) Render(ctx context.Context) (string, error) {
	if m == nil {
		return "", fmt.Errorf("review metrics registry is unavailable")
	}
	collectCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	text, err := m.collect(collectCtx)
	if err != nil {
		m.sqliteBusy.Add(1)
		m.mu.RLock()
		cached := m.cached
		m.mu.RUnlock()
		if cached != "" {
			return cached, nil
		}
		return "", err
	}
	m.mu.Lock()
	m.cached = text
	m.mu.Unlock()
	return text, nil
}

func (m *ReviewMetrics) collect(ctx context.Context) (string, error) {
	now := m.Now().UTC()
	snapshot := m.Status.Snapshot()
	lines := []string{"# TYPE review_service_ready gauge"}
	ready := 0
	if (snapshot.State == ReviewServiceReady || snapshot.State == ReviewServiceDegraded) && snapshot.Admission == ReviewAdmissionOpen {
		ready = 1
	}
	lines = append(lines, fmt.Sprintf("review_service_ready %d", ready))
	if started, err := time.Parse(time.RFC3339Nano, snapshot.StartedAt); err == nil {
		lines = append(lines, fmt.Sprintf("review_service_uptime_seconds %.3f", maxReviewMetricFloat(0, now.Sub(started).Seconds())))
	} else {
		lines = append(lines, "review_service_uptime_seconds 0")
	}
	for _, component := range snapshot.Components {
		up := 0
		if component.Status == ReviewComponentRunning || component.Status == ReviewComponentIdle {
			up = 1
		}
		label := reviewMetricLabels(map[string]string{"component": component.Component})
		lines = append(lines, fmt.Sprintf("review_service_component_up%s %d", label, up))
		age := 0.0
		if heartbeat, err := time.Parse(time.RFC3339Nano, component.LastHeartbeatAt); err == nil {
			age = maxReviewMetricFloat(0, now.Sub(heartbeat).Seconds())
		}
		lines = append(lines, fmt.Sprintf("review_service_component_last_heartbeat_age_seconds%s %.3f", label, age))
	}
	for _, metric := range []string{"review_handler_latency_seconds", "review_gitlink_request_latency_seconds", "review_feishu_request_latency_seconds"} {
		lines = append(lines, "# TYPE "+metric+" histogram", metric+"_bucket{le=\"0.1\"} 0", metric+"_bucket{le=\"0.5\"} 0", metric+"_bucket{le=\"1\"} 0", metric+"_bucket{le=\"5\"} 0", metric+"_bucket{le=\"+Inf\"} 0", metric+"_sum 0", metric+"_count 0")
	}
	if m.Store == nil || m.Store.db == nil {
		return "", fmt.Errorf("review metrics SQLite store is unavailable")
	}
	if err := m.appendDatabaseMetrics(ctx, &lines, now); err != nil {
		return "", err
	}
	lines = append(lines,
		fmt.Sprintf("review_job_retries_total %d", m.jobRetries.Load()),
		fmt.Sprintf("review_job_coalesced_total %d", m.jobCoalesced.Load()),
		fmt.Sprintf("review_job_admission_rejected_total %d", m.admissionRejects.Load()),
		fmt.Sprintf("review_rate_limit_rejected_total %d", m.rateRejects.Load()),
		fmt.Sprintf("review_sqlite_busy_total %d", m.sqliteBusy.Load()),
		fmt.Sprintf("review_backup_last_success_timestamp_seconds %d", m.backupSuccess.Load()),
		fmt.Sprintf("review_backup_last_failure_timestamp_seconds %d", m.backupFailure.Load()),
		fmt.Sprintf("review_retention_last_success_timestamp_seconds %d", m.retentionSuccess.Load()),
		fmt.Sprintf("review_wal_checkpoint_last_success_timestamp_seconds %d", m.walSuccess.Load()),
	)
	for _, path := range []struct{ metric, file string }{{"review_sqlite_file_size_bytes", m.DBPath}, {"review_sqlite_wal_size_bytes", m.DBPath + "-wal"}} {
		size := int64(0)
		if info, err := os.Stat(path.file); err == nil {
			size = info.Size()
		}
		lines = append(lines, fmt.Sprintf("%s %d", path.metric, size))
	}
	sort.Strings(lines[1:])
	return strings.Join(lines, "\n") + "\n", nil
}

func (m *ReviewMetrics) appendDatabaseMetrics(ctx context.Context, lines *[]string, now time.Time) error {
	queries := []struct {
		query  string
		metric string
		label1 string
		label2 string
	}{
		{`SELECT status,COUNT(*) FROM review_event_inbox WHERE status IN ('received','normalized','processing','routed') GROUP BY status`, "review_event_inbox_depth", "result", ""},
		{`SELECT queue_class,COUNT(*) FROM review_gateway_jobs WHERE status IN ('queued','running','retry_scheduled') GROUP BY queue_class`, "review_job_queue_depth", "queue_class", ""},
		{`SELECT operation_kind,status,COUNT(*) FROM review_operations GROUP BY operation_kind,status`, "review_operation_depth", "operation_kind", "operation_status"},
		{`SELECT resource_type,status,COUNT(*) FROM review_resource_projection_status GROUP BY resource_type,status`, "review_projection_status_total", "resource_type", "result"},
	}
	for _, spec := range queries {
		rows, err := m.Store.db.QueryContext(ctx, spec.query)
		if err != nil {
			return err
		}
		for rows.Next() {
			if spec.label2 == "" {
				var value string
				var count int
				if err := rows.Scan(&value, &count); err != nil {
					_ = rows.Close()
					return err
				}
				*lines = append(*lines, fmt.Sprintf("%s%s %d", spec.metric, reviewMetricLabels(map[string]string{spec.label1: value}), count))
			} else {
				var first, second string
				var count int
				if err := rows.Scan(&first, &second, &count); err != nil {
					_ = rows.Close()
					return err
				}
				*lines = append(*lines, fmt.Sprintf("%s%s %d", spec.metric, reviewMetricLabels(map[string]string{spec.label1: first, spec.label2: second}), count))
			}
		}
		_ = rows.Close()
	}
	for _, item := range []struct{ metric, query string }{
		{"review_operation_attempts_total", `SELECT COUNT(*) FROM review_operation_attempts`},
		{"review_operation_unknown_total", `SELECT COUNT(*) FROM review_operations WHERE status IN ('unknown','needs_reconciliation')`},
		{"review_operation_dead_letter_total", `SELECT COUNT(*) FROM review_dead_letters WHERE status='open'`},
		{"review_reconciliation_pending", `SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE status IN ('pending','checking')`},
		{"review_reconciliation_manual_required", `SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE status='manual_required'`},
	} {
		var count int
		if err := m.Store.db.QueryRowContext(ctx, item.query).Scan(&count); err != nil {
			return err
		}
		*lines = append(*lines, fmt.Sprintf("%s %d", item.metric, count))
	}
	for _, item := range []struct{ metric, query string }{
		{"review_event_oldest_age_seconds", `SELECT COALESCE(MIN(received_at),'') FROM review_event_inbox WHERE status IN ('received','normalized','processing','routed')`},
		{"review_job_oldest_age_seconds", `SELECT COALESCE(MIN(created_at),'') FROM review_gateway_jobs WHERE status IN ('queued','running','retry_scheduled')`},
		{"review_operation_oldest_age_seconds", `SELECT COALESCE(MIN(created_at),'') FROM review_operations WHERE status IN ('pending','leased','writing','retry_scheduled','blocked')`},
	} {
		var oldest string
		if err := m.Store.db.QueryRowContext(ctx, item.query).Scan(&oldest); err != nil {
			return err
		}
		age := 0.0
		if parsed, err := time.Parse(time.RFC3339Nano, oldest); err == nil {
			age = maxReviewMetricFloat(0, now.Sub(parsed).Seconds())
		}
		*lines = append(*lines, fmt.Sprintf("%s %.3f", item.metric, age))
	}
	return nil
}

func reviewMetricLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		if reviewMetricAllowedLabels[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n").Replace(labels[key])
		values = append(values, key+"=\""+value+"\"")
	}
	if len(values) == 0 {
		return ""
	}
	return "{" + strings.Join(values, ",") + "}"
}

func maxReviewMetricFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func reviewMetricFloat(value float64) string { return strconv.FormatFloat(value, 'f', 3, 64) }
