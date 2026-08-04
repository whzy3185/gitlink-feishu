package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	reviewGatewayDefaultMaxAttempts = 3
	reviewGatewayReplyMaxAttempts   = 3
	reviewGatewaySQLiteBusyMS       = 400
)

const reviewGatewayStateSchema = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS review_gateway_events (
    dedupe_key TEXT PRIMARY KEY,
    received_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_gateway_events_expires_at
    ON review_gateway_events(expires_at);

CREATE TABLE IF NOT EXISTS review_gateway_jobs (
    job_id TEXT PRIMARY KEY,
    dedupe_key TEXT NOT NULL,
    status TEXT NOT NULL,
    action TEXT NOT NULL,
    repository TEXT,
    pr_number INTEGER,
    requested_by TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    error_summary TEXT NOT NULL DEFAULT '',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_attempt_at TEXT NOT NULL DEFAULT '',
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    result_json TEXT NOT NULL DEFAULT '',
    handler_latency_ms INTEGER NOT NULL DEFAULT 0,
    reply_status TEXT NOT NULL DEFAULT 'none',
    reply_attempt_count INTEGER NOT NULL DEFAULT 0,
    reply_next_attempt_at TEXT NOT NULL DEFAULT '',
    reply_lease_owner TEXT NOT NULL DEFAULT '',
    reply_lease_expires_at TEXT NOT NULL DEFAULT '',
    reply_message_id TEXT NOT NULL DEFAULT '',
    reply_error_summary TEXT NOT NULL DEFAULT '',
    reply_requires_reconciliation INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS review_gateway_jobs_status
    ON review_gateway_jobs(status);

CREATE TABLE IF NOT EXISTS review_collaboration_items (
    pr_key TEXT PRIMARY KEY,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    chat_id TEXT NOT NULL,
    review_stage TEXT NOT NULL DEFAULT 'unreviewed',
    decision TEXT NOT NULL DEFAULT 'pending',
    collection_status TEXT NOT NULL DEFAULT 'pending',
    head_sha TEXT NOT NULL DEFAULT '',
    source_fingerprint TEXT NOT NULL DEFAULT '',
    assigned_to TEXT NOT NULL DEFAULT '',
    collaboration_status TEXT NOT NULL DEFAULT 'unassigned',
    due_at TEXT NOT NULL DEFAULT '',
    archived INTEGER NOT NULL DEFAULT 0,
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_collaboration_items_assignee
    ON review_collaboration_items(assigned_to, archived);

CREATE TABLE IF NOT EXISTS review_pr_snapshots (
    snapshot_key TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    review_stage TEXT NOT NULL DEFAULT 'unreviewed',
    decision TEXT NOT NULL DEFAULT 'pending',
    collection_status TEXT NOT NULL DEFAULT 'pending',
    head_sha TEXT NOT NULL DEFAULT '',
    source_fingerprint TEXT NOT NULL DEFAULT '',
    gitlink_state TEXT NOT NULL DEFAULT '',
    archived INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    UNIQUE (installation_id, repository, pr_number)
);
CREATE INDEX IF NOT EXISTS review_pr_snapshots_repository
    ON review_pr_snapshots(installation_id, repository, updated_at);

CREATE TABLE IF NOT EXISTS review_pr_presentations (
    presentation_key TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    schema_version TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    base_branch TEXT NOT NULL DEFAULT '',
    head_branch TEXT NOT NULL DEFAULT '',
    head_sha TEXT NOT NULL DEFAULT '',
    patchset_id TEXT NOT NULL DEFAULT '',
    gitlink_state TEXT NOT NULL DEFAULT '',
    files_count INTEGER NOT NULL DEFAULT 0,
    commits_count INTEGER NOT NULL DEFAULT 0,
    additions INTEGER NOT NULL DEFAULT 0,
    deletions INTEGER NOT NULL DEFAULT 0,
    review_stage TEXT NOT NULL DEFAULT '',
    decision TEXT NOT NULL DEFAULT '',
    review_count INTEGER NOT NULL DEFAULT 0,
    thread_count INTEGER NOT NULL DEFAULT 0,
    open_thread_count INTEGER NOT NULL DEFAULT 0,
    risk_level TEXT NOT NULL DEFAULT '',
    collection_status TEXT NOT NULL DEFAULT '',
    partial INTEGER NOT NULL DEFAULT 0,
    reviewers_json TEXT NOT NULL DEFAULT '[]',
    unknowns_json TEXT NOT NULL DEFAULT '[]',
    recommended_next_step TEXT NOT NULL DEFAULT '',
    gitlink_url TEXT NOT NULL DEFAULT '',
    source_fingerprint TEXT NOT NULL DEFAULT '',
    content_fingerprint TEXT NOT NULL DEFAULT '',
    source_completed_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    UNIQUE (installation_id, repository, pr_number)
);
CREATE INDEX IF NOT EXISTS review_pr_presentations_repository
    ON review_pr_presentations(installation_id, repository, updated_at);

CREATE TABLE IF NOT EXISTS chat_pr_presentations (
    presentation_key TEXT PRIMARY KEY,
    app_scope TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    canonical_message_id TEXT NOT NULL DEFAULT '',
    content_fingerprint TEXT NOT NULL DEFAULT '',
    source_completed_at TEXT NOT NULL DEFAULT '',
    presentation_version INTEGER NOT NULL DEFAULT 0,
    card_status TEXT NOT NULL DEFAULT 'pending',
    last_operation_id TEXT NOT NULL DEFAULT '',
    last_patch_at TEXT NOT NULL DEFAULT '',
    requires_reconciliation INTEGER NOT NULL DEFAULT 0,
    last_error_summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (app_scope, installation_id, chat_id, repository, pr_number)
);
CREATE INDEX IF NOT EXISTS chat_pr_presentations_status
    ON chat_pr_presentations(card_status, requires_reconciliation, updated_at);

CREATE TABLE IF NOT EXISTS review_collaboration_states (
    collaboration_key TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    assigned_to TEXT NOT NULL DEFAULT '',
    assigned_display_name TEXT NOT NULL DEFAULT '',
    collaboration_status TEXT NOT NULL DEFAULT 'unassigned',
    due_at TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    UNIQUE (installation_id, chat_id, repository, pr_number)
);
CREATE INDEX IF NOT EXISTS review_collaboration_states_assignee
    ON review_collaboration_states(installation_id, chat_id, assigned_to, collaboration_status);

CREATE TABLE IF NOT EXISTS review_collaboration_audit (
    audit_id TEXT PRIMARY KEY,
    pr_key TEXT NOT NULL,
    action TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    before_json TEXT NOT NULL,
    after_json TEXT NOT NULL,
    source_job_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_collaboration_audit_pr
    ON review_collaboration_audit(pr_key, created_at);

CREATE TABLE IF NOT EXISTS review_action_plans (
    plan_id TEXT PRIMARY KEY,
	installation_id TEXT NOT NULL,
	source_chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    actor_id TEXT NOT NULL,
    gitlink_login TEXT NOT NULL,
    expected_head_sha TEXT NOT NULL,
    source_fingerprint TEXT NOT NULL,
    review_status TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    source_job_id TEXT NOT NULL,
    review_id TEXT NOT NULL DEFAULT '',
    error_summary TEXT NOT NULL DEFAULT '',
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    reconciliation_status TEXT NOT NULL DEFAULT 'not_required',
    mutation_status TEXT NOT NULL DEFAULT 'none',
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_action_plans_status
    ON review_action_plans(status, expires_at);

CREATE TABLE IF NOT EXISTS review_collaboration_resources (
    work_item_key TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    remote_id TEXT NOT NULL DEFAULT '',
    content_fingerprint TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (work_item_key, resource_type)
);

CREATE TABLE IF NOT EXISTS review_resource_scope_policies (
    installation_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    target_scope TEXT NOT NULL,
    migration_enabled INTEGER NOT NULL DEFAULT 0,
    revision INTEGER NOT NULL DEFAULT 1,
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (installation_id, resource_type),
    CHECK (resource_type IN ('feishu_bitable', 'feishu_doc')),
    CHECK (target_scope IN ('chat', 'installation', 'disabled'))
);

CREATE TABLE IF NOT EXISTS review_resource_migrations (
    migration_id TEXT PRIMARY KEY,
    legacy_work_item_key TEXT NOT NULL,
    target_work_item_key TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL,
    target_scope TEXT NOT NULL DEFAULT '',
    installation_id TEXT NOT NULL DEFAULT '',
    repository TEXT NOT NULL DEFAULT '',
    pr_number INTEGER NOT NULL DEFAULT 0,
    chat_id_hash TEXT NOT NULL DEFAULT '',
    remote_id_hash TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    reason_code TEXT NOT NULL DEFAULT '',
    verification_method TEXT NOT NULL DEFAULT '',
    verified_by_hash TEXT NOT NULL DEFAULT '',
    verified_at TEXT NOT NULL DEFAULT '',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    error_summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_resource_migrations_status
    ON review_resource_migrations(status, resource_type, updated_at);

CREATE TABLE IF NOT EXISTS review_resource_migration_audit (
    audit_id TEXT PRIMARY KEY,
    migration_id TEXT NOT NULL,
    action TEXT NOT NULL,
    from_status TEXT NOT NULL,
    to_status TEXT NOT NULL,
    actor_hash TEXT NOT NULL DEFAULT '',
    detail_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_resource_migration_audit_migration
    ON review_resource_migration_audit(migration_id, created_at);
CREATE TRIGGER IF NOT EXISTS review_resource_migration_audit_no_update
    BEFORE UPDATE ON review_resource_migration_audit
    BEGIN SELECT RAISE(ABORT, 'review resource migration audit is append-only'); END;
CREATE TRIGGER IF NOT EXISTS review_resource_migration_audit_no_delete
    BEFORE DELETE ON review_resource_migration_audit
    BEGIN SELECT RAISE(ABORT, 'review resource migration audit is append-only'); END;

CREATE TABLE IF NOT EXISTS gitlink_installations (
    installation_id TEXT PRIMARY KEY,
    gitlink_host TEXT NOT NULL,
    owner TEXT NOT NULL DEFAULT '',
    credential_ref TEXT NOT NULL DEFAULT '',
    operation_mode TEXT NOT NULL,
	allow_public_read INTEGER NOT NULL DEFAULT 0,
    webhook_id TEXT NOT NULL DEFAULT '',
    webhook_secret_ref TEXT NOT NULL DEFAULT '',
    webhook_signature_mode TEXT NOT NULL DEFAULT 'body_sha256',
    webhook_timestamp_mode TEXT NOT NULL DEFAULT 'optional',
    webhook_max_skew_seconds INTEGER NOT NULL DEFAULT 300,
    webhook_delivery_required INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS installation_repositories (
    installation_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (installation_id, repository),
    FOREIGN KEY (installation_id) REFERENCES gitlink_installations(installation_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chat_repository_bindings (
    chat_id TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 0,
	allow_public_read INTEGER NOT NULL DEFAULT 0,
    admin_user_ids_json TEXT NOT NULL DEFAULT '[]',
    allowed_user_ids_json TEXT NOT NULL DEFAULT '[]',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (chat_id, installation_id, repository),
    FOREIGN KEY (installation_id) REFERENCES gitlink_installations(installation_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS review_chat_subscriptions (
    subscription_id TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pull_events INTEGER NOT NULL DEFAULT 0,
    review_events INTEGER NOT NULL DEFAULT 0,
    thread_events INTEGER NOT NULL DEFAULT 0,
    merge_events INTEGER NOT NULL DEFAULT 0,
    ci_events INTEGER NOT NULL DEFAULT 0,
    notification_mode TEXT NOT NULL DEFAULT 'canonical_only',
    enabled INTEGER NOT NULL DEFAULT 0,
    revision INTEGER NOT NULL DEFAULT 1,
    created_by_hash TEXT NOT NULL DEFAULT '',
    updated_by_hash TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (installation_id, chat_id, repository)
);
CREATE INDEX IF NOT EXISTS review_chat_subscriptions_enabled
    ON review_chat_subscriptions(installation_id, repository, enabled, updated_at);

CREATE TABLE IF NOT EXISTS review_event_inbox (
    event_id TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    delivery_key TEXT NOT NULL,
    delivery_hash TEXT NOT NULL,
    source TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    event_type TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT '',
    repository TEXT NOT NULL DEFAULT '',
    pr_number INTEGER NOT NULL DEFAULT 0,
    head_sha TEXT NOT NULL DEFAULT '',
    actor_hash TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL DEFAULT '',
    received_at TEXT NOT NULL,
    signature_status TEXT NOT NULL,
    timestamp_status TEXT NOT NULL,
    delivery_status TEXT NOT NULL,
    payload_fingerprint TEXT NOT NULL,
    canonical_event_json TEXT NOT NULL DEFAULT '',
    sanitized_payload_json TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    reason_code TEXT NOT NULL DEFAULT '',
    route_revision INTEGER NOT NULL DEFAULT 0,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT NOT NULL DEFAULT '',
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    error_summary TEXT NOT NULL DEFAULT '',
    processed_at TEXT NOT NULL DEFAULT '',
    replay_of_event_id TEXT NOT NULL DEFAULT '',
    replay_request_key TEXT NOT NULL DEFAULT '',
    UNIQUE (installation_id, delivery_key)
);
CREATE INDEX IF NOT EXISTS review_event_inbox_status
    ON review_event_inbox(status, next_attempt_at, received_at);
CREATE UNIQUE INDEX IF NOT EXISTS review_event_inbox_replay_request
    ON review_event_inbox(replay_request_key) WHERE replay_request_key <> '';

CREATE TABLE IF NOT EXISTS review_event_routes (
    event_id TEXT NOT NULL,
    subscription_id TEXT NOT NULL,
    installation_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    subscription_revision INTEGER NOT NULL,
    notification_mode TEXT NOT NULL,
    job_id TEXT NOT NULL DEFAULT '',
    route_status TEXT NOT NULL,
    reason_code TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (event_id, subscription_id)
);

CREATE TABLE IF NOT EXISTS review_reconciliation_cursors (
    cursor_key TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    pr_number INTEGER NOT NULL,
    last_checked_at TEXT NOT NULL DEFAULT '',
    last_enqueued_at TEXT NOT NULL DEFAULT '',
    next_check_at TEXT NOT NULL DEFAULT '',
    last_source_fingerprint TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    error_summary TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    UNIQUE (installation_id, chat_id, repository, pr_number)
);

CREATE TABLE IF NOT EXISTS review_identity_bindings (
    installation_id TEXT NOT NULL,
    feishu_user_id TEXT NOT NULL,
    gitlink_login TEXT NOT NULL,
    verification_method TEXT NOT NULL,
    verified_at TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (installation_id, feishu_user_id),
    FOREIGN KEY (installation_id) REFERENCES gitlink_installations(installation_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS review_gateway_configuration_audit (
    audit_id TEXT PRIMARY KEY,
    config_fingerprint TEXT NOT NULL,
    source TEXT NOT NULL,
    installation_count INTEGER NOT NULL,
    binding_count INTEGER NOT NULL,
    repository_count INTEGER NOT NULL,
    applied_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS review_gateway_configuration_audit_applied
    ON review_gateway_configuration_audit(applied_at);
`

var reviewGatewayJobMigrations = map[string]string{
	"attempt_count":          "INTEGER NOT NULL DEFAULT 0",
	"max_attempts":           "INTEGER NOT NULL DEFAULT 3",
	"next_attempt_at":        "TEXT NOT NULL DEFAULT ''",
	"lease_owner":            "TEXT NOT NULL DEFAULT ''",
	"lease_expires_at":       "TEXT NOT NULL DEFAULT ''",
	"result_json":            "TEXT NOT NULL DEFAULT ''",
	"handler_latency_ms":     "INTEGER NOT NULL DEFAULT 0",
	"reply_status":           "TEXT NOT NULL DEFAULT 'none'",
	"reply_attempt_count":    "INTEGER NOT NULL DEFAULT 0",
	"reply_next_attempt_at":  "TEXT NOT NULL DEFAULT ''",
	"reply_lease_owner":      "TEXT NOT NULL DEFAULT ''",
	"reply_lease_expires_at": "TEXT NOT NULL DEFAULT ''",
	"reply_message_id":       "TEXT NOT NULL DEFAULT ''",
	"reply_error_summary":    "TEXT NOT NULL DEFAULT ''",
}

var reviewActionPlanMigrations = map[string]string{
	"installation_id":       "TEXT NOT NULL DEFAULT ''",
	"source_chat_id":        "TEXT NOT NULL DEFAULT ''",
	"lease_owner":           "TEXT NOT NULL DEFAULT ''",
	"lease_expires_at":      "TEXT NOT NULL DEFAULT ''",
	"attempt_count":         "INTEGER NOT NULL DEFAULT 0",
	"max_attempts":          "INTEGER NOT NULL DEFAULT 3",
	"reconciliation_status": "TEXT NOT NULL DEFAULT 'not_required'",
	"mutation_status":       "TEXT NOT NULL DEFAULT 'none'",
}

var reviewInstallationMigrations = map[string]string{
	"allow_public_read": "INTEGER NOT NULL DEFAULT 0",
}

var reviewChatBindingMigrations = map[string]string{
	"allow_public_read": "INTEGER NOT NULL DEFAULT 0",
}

var reviewCollaborationAuditMigrations = map[string]string{
	"installation_id": "TEXT NOT NULL DEFAULT ''",
	"chat_id":         "TEXT NOT NULL DEFAULT ''",
	"repository":      "TEXT NOT NULL DEFAULT ''",
	"pr_number":       "INTEGER NOT NULL DEFAULT 0",
}

var (
	reviewGatewayErrorURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)
	reviewGatewayErrorRules      = []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{
			pattern:     regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s,;]+`),
			replacement: "${1}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(cookie|set-cookie|x-auth-token|private-token|access_token|refresh_token|client_secret|app_secret)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(token|secret|password)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
	}
)

type ReviewGatewayClaimOptions struct {
	LeaseOwner    string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
	QueueClass    string
}

type ReviewGatewayJobOutcome struct {
	Job       ReviewGatewayJob
	Result    ReviewGatewayExecutionResult
	Err       error
	WillRetry bool
}

type ReviewGatewayPendingReply struct {
	Job          ReviewGatewayJob
	Result       ReviewGatewayExecutionResult
	AttemptCount int
}

type ReviewGatewayJobStore interface {
	SaveJob(ctx context.Context, job ReviewGatewayJob) error
	ClaimReadyJobs(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error)
	CompleteJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error
	RetryOrFailJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error)
	RecordHandlerLatency(ctx context.Context, jobID string, latencyMs int64) error
	ClaimPendingReplies(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error)
	MarkReplySent(ctx context.Context, jobID, messageID string, now time.Time) error
	MarkReplyUnknown(ctx context.Context, jobID, messageID, errorSummary string, now time.Time) error
	MarkReplyFailed(ctx context.Context, pending ReviewGatewayPendingReply, errorSummary string, now time.Time) (bool, error)
	UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error
}

type ReviewGatewayStateStore interface {
	ReviewGatewayDeduper
	ReviewGatewayJobStore
	Close() error
}

type MemoryReviewGatewayJobStore struct {
	mu            sync.Mutex
	Jobs          map[string]ReviewGatewayJob
	Status        map[string]string
	Errors        map[string]string
	Results       map[string]ReviewGatewayExecutionResult
	ReplyStatus   map[string]string
	ReplyAttempts map[string]int
	ReplyLeases   map[string]string
}

type SQLiteReviewGatewayStore struct {
	db   *sql.DB
	path string
}

type reviewGatewayLatencyObservation struct {
	JobID     string
	LatencyMs int64
}

type reviewGatewaySchemaMigration struct {
	Version    int
	Name       string
	Columns    map[string]map[string]string
	Statements []string
}

var reviewGatewaySchemaMigrations = []reviewGatewaySchemaMigration{
	{
		Version: 1,
		Name:    "review_pr_presentations_v1",
		Columns: map[string]map[string]string{
			"review_pr_presentations": {"source_completed_at": "TEXT NOT NULL DEFAULT ''"},
		},
		Statements: []string{
			`SELECT presentation_key, source_completed_at FROM review_pr_presentations LIMIT 0`,
		},
	},
	{
		Version: 2,
		Name:    "chat_pr_presentations_v1",
		Columns: map[string]map[string]string{
			"chat_pr_presentations": {
				"last_operation_id":   "TEXT NOT NULL DEFAULT ''",
				"last_error_summary":  "TEXT NOT NULL DEFAULT ''",
				"source_completed_at": "TEXT NOT NULL DEFAULT ''",
			},
		},
		Statements: []string{
			`SELECT presentation_key, last_operation_id, last_error_summary, source_completed_at FROM chat_pr_presentations LIMIT 0`,
		},
	},
	{
		Version: 3,
		Name:    "reply_unknown_state_v1",
		Columns: map[string]map[string]string{
			"review_gateway_jobs": {"reply_requires_reconciliation": "INTEGER NOT NULL DEFAULT 0"},
		},
		Statements: []string{
			`SELECT reply_requires_reconciliation FROM review_gateway_jobs LIMIT 0`,
		},
	},
	{
		Version: 4,
		Name:    "review_resource_scope_policies_v1",
		Statements: []string{
			`SELECT installation_id, resource_type, target_scope, migration_enabled FROM review_resource_scope_policies LIMIT 0`,
		},
	},
	{
		Version: 5,
		Name:    "review_resource_migrations_v1",
		Statements: []string{
			`SELECT migration_id, legacy_work_item_key, target_work_item_key, status FROM review_resource_migrations LIMIT 0`,
		},
	},
	{
		Version: 6,
		Name:    "review_resource_migration_audit_v1",
		Statements: []string{
			`SELECT audit_id, migration_id, action, from_status, to_status FROM review_resource_migration_audit LIMIT 0`,
			`CREATE TRIGGER IF NOT EXISTS review_resource_migration_audit_no_update
			 BEFORE UPDATE ON review_resource_migration_audit
			 BEGIN SELECT RAISE(ABORT, 'review resource migration audit is append-only'); END`,
			`CREATE TRIGGER IF NOT EXISTS review_resource_migration_audit_no_delete
			 BEFORE DELETE ON review_resource_migration_audit
			 BEGIN SELECT RAISE(ABORT, 'review resource migration audit is append-only'); END`,
		},
	},
	{
		Version: 7,
		Name:    "review_chat_subscriptions_v1",
		Statements: []string{
			`SELECT subscription_id, installation_id, chat_id, repository,
			 pull_events, review_events, thread_events, merge_events, ci_events,
			 notification_mode, enabled, revision FROM review_chat_subscriptions LIMIT 0`,
		},
	},
	{
		Version: 8,
		Name:    "review_event_inbox_routes_v1",
		Statements: []string{
			`SELECT event_id, installation_id, delivery_key, canonical_event_json,
			 status, route_revision, lease_owner, replay_request_key FROM review_event_inbox LIMIT 0`,
			`SELECT event_id, subscription_id, subscription_revision, job_id,
			 route_status FROM review_event_routes LIMIT 0`,
		},
	},
	{
		Version: 9,
		Name:    "gitlink_webhook_security_policy_v1",
		Columns: map[string]map[string]string{
			"gitlink_installations": {
				"webhook_signature_mode":    "TEXT NOT NULL DEFAULT 'body_sha256'",
				"webhook_timestamp_mode":    "TEXT NOT NULL DEFAULT 'optional'",
				"webhook_max_skew_seconds":  "INTEGER NOT NULL DEFAULT 300",
				"webhook_delivery_required": "INTEGER NOT NULL DEFAULT 0",
			},
		},
		Statements: []string{
			`SELECT webhook_signature_mode, webhook_timestamp_mode,
			 webhook_max_skew_seconds, webhook_delivery_required FROM gitlink_installations LIMIT 0`,
		},
	},
	{
		Version: 10,
		Name:    "review_reconciliation_cursors_v1",
		Statements: []string{
			`SELECT cursor_key, installation_id, chat_id, repository, pr_number,
			 last_checked_at, last_enqueued_at, next_check_at, last_source_fingerprint,
			 status FROM review_reconciliation_cursors LIMIT 0`,
		},
	},
	{
		Version: 11,
		Name:    "review_operations_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_operations (
				operation_id TEXT PRIMARY KEY,
				operation_kind TEXT NOT NULL,
				queue_class TEXT NOT NULL,
				installation_id TEXT NOT NULL DEFAULT '',
				chat_id TEXT NOT NULL DEFAULT '',
				repository TEXT NOT NULL DEFAULT '',
				pr_number INTEGER NOT NULL DEFAULT 0,
				source_job_id TEXT NOT NULL DEFAULT '',
				source_event_id TEXT NOT NULL DEFAULT '',
				consumer_id TEXT NOT NULL DEFAULT '',
				work_item_key TEXT NOT NULL DEFAULT '',
				resource_type TEXT NOT NULL DEFAULT '',
				idempotency_key TEXT NOT NULL UNIQUE,
				desired_fingerprint TEXT NOT NULL DEFAULT '',
				applied_fingerprint TEXT NOT NULL DEFAULT '',
				expected_remote_id TEXT NOT NULL DEFAULT '',
				remote_id TEXT NOT NULL DEFAULT '',
				desired_json TEXT NOT NULL DEFAULT '{}',
				dependency_operation_id TEXT NOT NULL DEFAULT '',
				dependency_policy TEXT NOT NULL DEFAULT 'none',
				retry_safety TEXT NOT NULL,
				status TEXT NOT NULL,
				error_class TEXT NOT NULL DEFAULT '',
				error_code TEXT NOT NULL DEFAULT '',
				error_summary TEXT NOT NULL DEFAULT '',
				http_status INTEGER NOT NULL DEFAULT 0,
				mutation_status TEXT NOT NULL DEFAULT 'not_started',
				requires_reconciliation INTEGER NOT NULL DEFAULT 0,
				attempt_count INTEGER NOT NULL DEFAULT 0,
				max_attempts INTEGER NOT NULL DEFAULT 3,
				next_attempt_at TEXT NOT NULL DEFAULT '',
				retry_after_at TEXT NOT NULL DEFAULT '',
				lease_owner TEXT NOT NULL DEFAULT '',
				lease_expires_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				completed_at TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS review_operations_ready
			 ON review_operations(queue_class, status, next_attempt_at, created_at)`,
			`CREATE INDEX IF NOT EXISTS review_operations_scope
			 ON review_operations(installation_id, chat_id, repository, pr_number, operation_kind, updated_at)`,
		},
	},
	{
		Version: 12,
		Name:    "review_operation_attempts_projection_status_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_operation_attempts (
				attempt_id TEXT PRIMARY KEY,
				operation_id TEXT NOT NULL,
				attempt_number INTEGER NOT NULL,
				lease_owner_hash TEXT NOT NULL DEFAULT '',
				started_at TEXT NOT NULL,
				finished_at TEXT NOT NULL DEFAULT '',
				mutation_status TEXT NOT NULL DEFAULT 'not_started',
				error_class TEXT NOT NULL DEFAULT '',
				error_code TEXT NOT NULL DEFAULT '',
				error_summary TEXT NOT NULL DEFAULT '',
				http_status INTEGER NOT NULL DEFAULT 0,
				retry_after_at TEXT NOT NULL DEFAULT '',
				result_fingerprint TEXT NOT NULL DEFAULT '',
				UNIQUE(operation_id, attempt_number)
			)`,
			`CREATE INDEX IF NOT EXISTS review_operation_attempts_operation
			 ON review_operation_attempts(operation_id, attempt_number)`,
			`CREATE TRIGGER IF NOT EXISTS review_operation_attempts_finished_no_update
			 BEFORE UPDATE ON review_operation_attempts WHEN OLD.finished_at <> ''
			 BEGIN SELECT RAISE(ABORT, 'finished review operation attempt is append-only'); END`,
			`CREATE TRIGGER IF NOT EXISTS review_operation_attempts_no_delete
			 BEFORE DELETE ON review_operation_attempts
			 BEGIN SELECT RAISE(ABORT, 'review operation attempt is append-only'); END`,
			`CREATE TABLE IF NOT EXISTS review_resource_projection_status (
				projection_key TEXT PRIMARY KEY,
				installation_id TEXT NOT NULL,
				chat_id TEXT NOT NULL DEFAULT '',
				repository TEXT NOT NULL,
				pr_number INTEGER NOT NULL,
				resource_type TEXT NOT NULL,
				target_scope TEXT NOT NULL,
				desired_fingerprint TEXT NOT NULL DEFAULT '',
				applied_fingerprint TEXT NOT NULL DEFAULT '',
				operation_id TEXT NOT NULL DEFAULT '',
				status TEXT NOT NULL,
				error_class TEXT NOT NULL DEFAULT '',
				error_summary TEXT NOT NULL DEFAULT '',
				requires_reconciliation INTEGER NOT NULL DEFAULT 0,
				updated_at TEXT NOT NULL,
				UNIQUE(installation_id, chat_id, repository, pr_number, resource_type, target_scope)
			)`,
			`CREATE INDEX IF NOT EXISTS review_resource_projection_status_scope
			 ON review_resource_projection_status(installation_id, chat_id, repository, pr_number, resource_type)`,
		},
	},
	{
		Version: 13,
		Name:    "review_dead_letters_operation_reconciliation_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_dead_letters (
				dead_letter_id TEXT PRIMARY KEY,
				entity_type TEXT NOT NULL,
				entity_id TEXT NOT NULL,
				queue_class TEXT NOT NULL DEFAULT '',
				installation_id TEXT NOT NULL DEFAULT '',
				chat_id_hash TEXT NOT NULL DEFAULT '',
				repository TEXT NOT NULL DEFAULT '',
				pr_number INTEGER NOT NULL DEFAULT 0,
				error_class TEXT NOT NULL,
				error_code TEXT NOT NULL DEFAULT '',
				error_summary TEXT NOT NULL DEFAULT '',
				attempt_count INTEGER NOT NULL DEFAULT 0,
				status TEXT NOT NULL DEFAULT 'open',
				resolution_action TEXT NOT NULL DEFAULT '',
				resolved_by_hash TEXT NOT NULL DEFAULT '',
				resolved_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				UNIQUE(entity_type, entity_id)
			)`,
			`CREATE INDEX IF NOT EXISTS review_dead_letters_status
			 ON review_dead_letters(status, entity_type, updated_at)`,
			`CREATE TABLE IF NOT EXISTS review_operation_reconciliation_tasks (
				reconciliation_id TEXT PRIMARY KEY,
				operation_id TEXT NOT NULL UNIQUE,
				resource_type TEXT NOT NULL,
				reason_code TEXT NOT NULL,
				status TEXT NOT NULL,
				verification_method TEXT NOT NULL DEFAULT '',
				attempt_count INTEGER NOT NULL DEFAULT 0,
				max_attempts INTEGER NOT NULL DEFAULT 3,
				next_attempt_at TEXT NOT NULL DEFAULT '',
				lease_owner TEXT NOT NULL DEFAULT '',
				lease_expires_at TEXT NOT NULL DEFAULT '',
				result_summary TEXT NOT NULL DEFAULT '',
				error_summary TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				resolved_at TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS review_operation_reconciliation_ready
			 ON review_operation_reconciliation_tasks(status, next_attempt_at, created_at)`,
		},
	},
	{
		Version: 14,
		Name:    "review_job_queue_classes_coalescing_v1",
		Columns: map[string]map[string]string{
			"review_gateway_jobs": {
				"queue_class":                     "TEXT NOT NULL DEFAULT 'gitlink_read'",
				"priority":                        "INTEGER NOT NULL DEFAULT 50",
				"coalesce_key":                    "TEXT NOT NULL DEFAULT ''",
				"coalesce_until":                  "TEXT NOT NULL DEFAULT ''",
				"operation_plan_status":           "TEXT NOT NULL DEFAULT 'none'",
				"operation_plan_attempt_count":    "INTEGER NOT NULL DEFAULT 0",
				"operation_plan_next_attempt_at":  "TEXT NOT NULL DEFAULT ''",
				"operation_plan_lease_owner":      "TEXT NOT NULL DEFAULT ''",
				"operation_plan_lease_expires_at": "TEXT NOT NULL DEFAULT ''",
				"operation_plan_error_summary":    "TEXT NOT NULL DEFAULT ''",
			},
		},
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_job_consumers (
				consumer_id TEXT PRIMARY KEY,
				job_id TEXT NOT NULL,
				source_type TEXT NOT NULL,
				chat_id TEXT NOT NULL,
				source_message_id TEXT NOT NULL DEFAULT '',
				requested_by_hash TEXT NOT NULL DEFAULT '',
				notification_mode TEXT NOT NULL DEFAULT 'canonical_only',
				status TEXT NOT NULL DEFAULT 'pending',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				UNIQUE(job_id, source_type, chat_id, source_message_id, notification_mode)
			)`,
			`CREATE INDEX IF NOT EXISTS review_gateway_jobs_queue_ready
			 ON review_gateway_jobs(queue_class, status, priority, next_attempt_at, created_at)`,
			`CREATE INDEX IF NOT EXISTS review_gateway_jobs_coalesce
			 ON review_gateway_jobs(coalesce_key, status, coalesce_until)`,
			`CREATE INDEX IF NOT EXISTS review_job_consumers_job
			 ON review_job_consumers(job_id, status, created_at)`,
		},
	},
	{
		Version: 15,
		Name:    "review_rate_limit_buckets_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_rate_limit_buckets (
				bucket_key TEXT PRIMARY KEY,
				bucket_type TEXT NOT NULL,
				subject_hash TEXT NOT NULL,
				window_started_at TEXT NOT NULL,
				request_count INTEGER NOT NULL DEFAULT 0,
				updated_at TEXT NOT NULL,
				UNIQUE(bucket_type, subject_hash, window_started_at)
			)`,
			`CREATE INDEX IF NOT EXISTS review_rate_limit_buckets_window
			 ON review_rate_limit_buckets(bucket_type, window_started_at, updated_at)`,
		},
	},
	{
		Version: 16,
		Name:    "review_service_instances_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_service_instances (
				instance_id TEXT PRIMARY KEY,
				process_id INTEGER NOT NULL DEFAULT 0,
				hostname_hash TEXT NOT NULL DEFAULT '',
				service_version TEXT NOT NULL DEFAULT '',
				commit_sha TEXT NOT NULL DEFAULT '',
				schema_version INTEGER NOT NULL DEFAULT 0,
				config_revision INTEGER NOT NULL DEFAULT 0,
				config_fingerprint TEXT NOT NULL DEFAULT '',
				status TEXT NOT NULL,
				admission_status TEXT NOT NULL,
				started_at TEXT NOT NULL,
				ready_at TEXT NOT NULL DEFAULT '',
				heartbeat_at TEXT NOT NULL DEFAULT '',
				shutdown_started_at TEXT NOT NULL DEFAULT '',
				stopped_at TEXT NOT NULL DEFAULT '',
				last_error_summary TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS review_service_instances_status
			 ON review_service_instances(status, heartbeat_at)`,
		},
	},
	{
		Version: 17,
		Name:    "review_configuration_revisions_v1",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS review_configuration_state (
				singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
				config_revision INTEGER NOT NULL DEFAULT 0,
				config_fingerprint TEXT NOT NULL DEFAULT '',
				source TEXT NOT NULL DEFAULT '',
				source_hash TEXT NOT NULL DEFAULT '',
				applied_by_hash TEXT NOT NULL DEFAULT '',
				applied_at TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE TABLE IF NOT EXISTS review_configuration_revisions (
				revision_id TEXT PRIMARY KEY,
				config_revision INTEGER NOT NULL,
				config_fingerprint TEXT NOT NULL,
				source TEXT NOT NULL,
				source_hash TEXT NOT NULL DEFAULT '',
				installation_count INTEGER NOT NULL DEFAULT 0,
				repository_count INTEGER NOT NULL DEFAULT 0,
				binding_count INTEGER NOT NULL DEFAULT 0,
				subscription_count INTEGER NOT NULL DEFAULT 0,
				identity_binding_count INTEGER NOT NULL DEFAULT 0,
				resource_policy_count INTEGER NOT NULL DEFAULT 0,
				worker_config_json TEXT NOT NULL DEFAULT '{}',
				limit_config_json TEXT NOT NULL DEFAULT '{}',
				service_config_json TEXT NOT NULL DEFAULT '{}',
				applied_by_hash TEXT NOT NULL DEFAULT '',
				applied_at TEXT NOT NULL
			)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS review_configuration_revisions_number
			 ON review_configuration_revisions(config_revision)`,
			`CREATE TRIGGER IF NOT EXISTS review_configuration_revisions_no_update
			 BEFORE UPDATE ON review_configuration_revisions
			 BEGIN SELECT RAISE(ABORT, 'review configuration revision is append-only'); END`,
			`CREATE TRIGGER IF NOT EXISTS review_configuration_revisions_no_delete
			 BEFORE DELETE ON review_configuration_revisions
			 BEGIN SELECT RAISE(ABORT, 'review configuration revision is append-only'); END`,
		},
	},
	{
		Version: 19,
		Name:    "review_configuration_entity_revisions_v1",
		Columns: map[string]map[string]string{
			"gitlink_installations": {
				"revision":        "INTEGER NOT NULL DEFAULT 1",
				"created_at":      "TEXT NOT NULL DEFAULT ''",
				"updated_by_hash": "TEXT NOT NULL DEFAULT ''",
			},
			"installation_repositories": {
				"revision":        "INTEGER NOT NULL DEFAULT 1",
				"created_at":      "TEXT NOT NULL DEFAULT ''",
				"updated_by_hash": "TEXT NOT NULL DEFAULT ''",
			},
			"chat_repository_bindings": {
				"revision":        "INTEGER NOT NULL DEFAULT 1",
				"created_at":      "TEXT NOT NULL DEFAULT ''",
				"updated_by_hash": "TEXT NOT NULL DEFAULT ''",
			},
			"review_identity_bindings": {
				"revision":        "INTEGER NOT NULL DEFAULT 1",
				"created_at":      "TEXT NOT NULL DEFAULT ''",
				"updated_by_hash": "TEXT NOT NULL DEFAULT ''",
			},
		},
		Statements: []string{
			`SELECT revision,created_at,updated_by_hash FROM gitlink_installations LIMIT 0`,
			`SELECT revision,created_at,updated_by_hash FROM installation_repositories LIMIT 0`,
			`SELECT revision,created_at,updated_by_hash FROM chat_repository_bindings LIMIT 0`,
			`SELECT revision,created_at,updated_by_hash FROM review_identity_bindings LIMIT 0`,
		},
	},
}

type ReviewGatewayQueue struct {
	gateway          *ReviewGateway
	store            ReviewGatewayJobStore
	wake             chan struct{}
	latencies        chan reviewGatewayLatencyObservation
	leaseOwner       string
	leaseDuration    time.Duration
	pollInterval     time.Duration
	onResult         func(ReviewGatewayJobOutcome)
	now              func() time.Time
	operationPlanner *ReviewOperationPlanner
	operationWake    func()
	limits           ReviewQueueLimits
}

func NewMemoryReviewGatewayJobStore() *MemoryReviewGatewayJobStore {
	return &MemoryReviewGatewayJobStore{
		Jobs:          map[string]ReviewGatewayJob{},
		Status:        map[string]string{},
		Errors:        map[string]string{},
		Results:       map[string]ReviewGatewayExecutionResult{},
		ReplyStatus:   map[string]string{},
		ReplyAttempts: map[string]int{},
		ReplyLeases:   map[string]string{},
	}
}

func (s *MemoryReviewGatewayJobStore) SaveJob(_ context.Context, job ReviewGatewayJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Jobs[job.JobID]; exists {
		return nil
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	s.Jobs[job.JobID] = job
	s.Status[job.JobID] = "queued"
	s.ReplyStatus[job.JobID] = "none"
	return nil
}

func (s *MemoryReviewGatewayJobStore) ClaimReadyJobs(_ context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalizeReviewGatewayClaimOptions(&opts)
	ids := make([]string, 0, len(s.Jobs))
	for id := range s.Jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []ReviewGatewayJob{}
	for _, id := range ids {
		if len(result) >= opts.Limit {
			break
		}
		job := s.Jobs[id]
		if opts.QueueClass != "" && job.QueueClass != opts.QueueClass {
			continue
		}
		status := s.Status[id]
		if status == "running" && reviewGatewayTimeDue(job.LeaseExpiresAt, opts.Now) {
			status = "queued"
			s.Status[id] = status
		}
		if status != "queued" || !reviewGatewayTimeDue(job.NextAttemptAt, opts.Now) || job.AttemptCount >= job.MaxAttempts {
			continue
		}
		job.AttemptCount++
		job.Status = "running"
		job.LeaseOwner = opts.LeaseOwner
		job.LeaseExpiresAt = opts.Now.Add(opts.LeaseDuration).UTC().Format(time.RFC3339Nano)
		s.Jobs[id] = job
		s.Status[id] = "running"
		result = append(result, job)
	}
	return result, nil
}

func (s *MemoryReviewGatewayJobStore) CompleteJob(_ context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Results[job.JobID] = result
	s.Status[job.JobID] = "completed"
	job.Status = "completed"
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	s.Jobs[job.JobID] = job
	if shouldDispatchReviewGatewayResult(job) {
		s.ReplyStatus[job.JobID] = "pending"
	}
	return nil
}

func (s *MemoryReviewGatewayJobStore) RetryOrFailJob(_ context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Results[job.JobID] = result
	s.Errors[job.JobID] = redactReviewGatewayError(errorSummary)
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	willRetry := job.AttemptCount < job.MaxAttempts
	if willRetry {
		job.Status = "queued"
		job.NextAttemptAt = now.Add(reviewGatewayRetryDelay(job.AttemptCount)).UTC().Format(time.RFC3339Nano)
		s.Status[job.JobID] = "queued"
	} else {
		job.Status = "failed"
		s.Status[job.JobID] = "failed"
		if shouldDispatchReviewGatewayResult(job) {
			s.ReplyStatus[job.JobID] = "pending"
		}
	}
	s.Jobs[job.JobID] = job
	return willRetry, nil
}

func (s *MemoryReviewGatewayJobStore) RecordHandlerLatency(_ context.Context, jobID string, latencyMs int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.Jobs[jobID]
	if !ok {
		return fmt.Errorf("review gateway job %q not found", jobID)
	}
	job.HandlerLatencyMs = latencyMs
	s.Jobs[jobID] = job
	return nil
}

func (s *MemoryReviewGatewayJobStore) ClaimPendingReplies(_ context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalizeReviewGatewayClaimOptions(&opts)
	ids := make([]string, 0, len(s.Jobs))
	for id := range s.Jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []ReviewGatewayPendingReply{}
	for _, id := range ids {
		if len(result) >= opts.Limit {
			break
		}
		status := s.ReplyStatus[id]
		job := s.Jobs[id]
		if status == "sending" && reviewGatewayTimeDue(s.ReplyLeases[id], opts.Now) {
			status = "pending"
			s.ReplyStatus[id] = status
		}
		if status != "pending" || s.ReplyAttempts[id] >= reviewGatewayReplyMaxAttempts {
			continue
		}
		s.ReplyStatus[id] = "sending"
		s.ReplyAttempts[id]++
		s.ReplyLeases[id] = opts.Now.Add(opts.LeaseDuration).UTC().Format(time.RFC3339Nano)
		result = append(result, ReviewGatewayPendingReply{
			Job:          job,
			Result:       s.Results[id],
			AttemptCount: s.ReplyAttempts[id],
		})
	}
	return result, nil
}

func (s *MemoryReviewGatewayJobStore) MarkReplySent(_ context.Context, jobID, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReplyStatus[jobID] = "sent"
	s.ReplyLeases[jobID] = ""
	return nil
}

func (s *MemoryReviewGatewayJobStore) MarkReplyUnknown(_ context.Context, jobID, _ string, errorSummary string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReplyStatus[jobID] = "unknown"
	s.ReplyLeases[jobID] = ""
	s.Errors[jobID] = redactReviewGatewayError(errorSummary)
	return nil
}

func (s *MemoryReviewGatewayJobStore) MarkReplyFailed(_ context.Context, pending ReviewGatewayPendingReply, errorSummary string, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors[pending.Job.JobID] = redactReviewGatewayError(errorSummary)
	willRetry := pending.AttemptCount < reviewGatewayReplyMaxAttempts
	if willRetry {
		s.ReplyStatus[pending.Job.JobID] = "pending"
	} else {
		s.ReplyStatus[pending.Job.JobID] = "failed"
	}
	s.ReplyLeases[pending.Job.JobID] = ""
	return willRetry, nil
}

func (s *MemoryReviewGatewayJobStore) UpdateJobStatus(_ context.Context, jobID, status, errorSummary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Jobs[jobID]; !ok {
		return fmt.Errorf("review gateway job %q not found", jobID)
	}
	s.Status[jobID] = status
	s.Errors[jobID] = redactReviewGatewayError(errorSummary)
	return nil
}

func OpenSQLiteReviewGatewayStore(path string) (*SQLiteReviewGatewayStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("review gateway state database path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve review gateway state database: %w", err)
	}
	if parent := filepath.Dir(absolute); parent != "." {
		if err := os.MkdirAll(parent, 0700); err != nil {
			return nil, fmt.Errorf("create review gateway state directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", absolute)
	if err != nil {
		return nil, fmt.Errorf("open review gateway state database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		fmt.Sprintf("PRAGMA busy_timeout=%d", reviewGatewaySQLiteBusyMS),
		reviewGatewayStateSchema,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize review gateway state database: %w", err)
		}
	}
	if err := ensureReviewGatewayJobColumns(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureReviewActionPlanColumns(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureReviewGatewayTableColumns(db, "gitlink_installations", reviewInstallationMigrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureReviewGatewayTableColumns(db, "chat_repository_bindings", reviewChatBindingMigrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureReviewGatewayTableColumns(db, "review_collaboration_audit", reviewCollaborationAuditMigrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := applyReviewGatewaySchemaMigrations(db, reviewGatewaySchemaMigrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateLegacyReviewCollaborationScope(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS review_gateway_jobs_ready
		 ON review_gateway_jobs(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS review_gateway_jobs_reply_ready
		 ON review_gateway_jobs(reply_status, reply_next_attempt_at)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize review gateway state index: %w", err)
		}
	}
	return &SQLiteReviewGatewayStore{db: db, path: absolute}, nil
}

func applyReviewGatewaySchemaMigrations(db *sql.DB, migrations []reviewGatewaySchemaMigration) error {
	for _, migration := range migrations {
		var applied int
		err := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version=?`, migration.Version).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inspect review gateway schema migration %d: %w", migration.Version, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin review gateway schema migration %d: %w", migration.Version, err)
		}
		tables := make([]string, 0, len(migration.Columns))
		for table := range migration.Columns {
			tables = append(tables, table)
		}
		sort.Strings(tables)
		for _, table := range tables {
			if err := ensureReviewGatewayTableColumnsTx(tx, table, migration.Columns[table]); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply review gateway schema migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
		for _, statement := range migration.Statements {
			if _, err := tx.Exec(statement); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply review gateway schema migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
			migration.Version, migration.Name, reviewGatewayTimestamp(time.Now().UTC()),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record review gateway schema migration %d: %w", migration.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit review gateway schema migration %d: %w", migration.Version, err)
		}
	}
	return nil
}

func ensureReviewGatewayTableColumnsTx(tx *sql.Tx, table string, migrations map[string]string) error {
	rows, err := tx.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read %s schema: %w", table, err)
		}
		existing[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s schema rows: %w", table, err)
	}
	names := make([]string, 0, len(migrations))
	for name := range migrations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existing[name] {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, migrations[name])); err != nil {
			return fmt.Errorf("migrate %s column %s: %w", table, name, err)
		}
	}
	return nil
}

func ensureReviewGatewayTableColumns(db *sql.DB, table string, migrations map[string]string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read %s schema: %w", table, err)
		}
		existing[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s schema rows: %w", table, err)
	}
	names := make([]string, 0, len(migrations))
	for name := range migrations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existing[name] {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, migrations[name])
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate %s column %s: %w", table, name, err)
		}
	}
	return nil
}

func ensureReviewGatewayJobColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(review_gateway_jobs)")
	if err != nil {
		return fmt.Errorf("inspect review gateway job schema: %w", err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read review gateway job schema: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close review gateway job schema rows: %w", err)
	}
	names := make([]string, 0, len(reviewGatewayJobMigrations))
	for name := range reviewGatewayJobMigrations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existing[name] {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE review_gateway_jobs ADD COLUMN %s %s", name, reviewGatewayJobMigrations[name])
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate review gateway job column %s: %w", name, err)
		}
	}
	return nil
}

func ensureReviewActionPlanColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(review_action_plans)")
	if err != nil {
		return fmt.Errorf("inspect review action plan schema: %w", err)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read review action plan schema: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close review action plan schema rows: %w", err)
	}
	names := make([]string, 0, len(reviewActionPlanMigrations))
	for name := range reviewActionPlanMigrations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if existing[name] {
			continue
		}
		statement := fmt.Sprintf(
			"ALTER TABLE review_action_plans ADD COLUMN %s %s",
			name,
			reviewActionPlanMigrations[name],
		)
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate review action plan column %s: %w", name, err)
		}
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) Reserve(key string, now time.Time, ttl time.Duration) (bool, error) {
	return s.ReserveContext(context.Background(), key, now, ttl)
}

func (s *SQLiteReviewGatewayStore) ReserveContext(ctx context.Context, key string, now time.Time, ttl time.Duration) (bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(key) == "" {
		return false, fmt.Errorf("review gateway state store and dedupe key are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin review gateway dedupe transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, _ = tx.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE expires_at <= ?", reviewGatewayTimestamp(now))
	result, err := tx.ExecContext(
		ctx,
		"INSERT OR IGNORE INTO review_gateway_events(dedupe_key, received_at, expires_at) VALUES(?, ?, ?)",
		key,
		reviewGatewayTimestamp(now),
		reviewGatewayTimestamp(now.Add(ttl)),
	)
	if err != nil {
		return false, fmt.Errorf("reserve review gateway event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read review gateway event reservation: %w", err)
	}
	if affected != 1 {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit review gateway event reservation: %w", err)
	}
	return true, nil
}

func (s *SQLiteReviewGatewayStore) Release(key string) {
	if s == nil || s.db == nil || strings.TrimSpace(key) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = s.db.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE dedupe_key = ?", key)
}

func (s *SQLiteReviewGatewayStore) SaveJob(ctx context.Context, job ReviewGatewayJob) error {
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode review gateway job: %w", err)
	}
	now := reviewGatewayTimestamp(time.Now())
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO review_gateway_jobs(
			job_id, dedupe_key, status, action, repository, pr_number,
			requested_by, chat_id, payload_json, created_at, updated_at,
			error_summary, attempt_count, max_attempts, next_attempt_at
		) VALUES(?, ?, 'queued', ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?)
		ON CONFLICT(job_id) DO NOTHING`,
		job.JobID,
		job.DedupeKey,
		job.Action,
		job.Repository,
		job.PRNumber,
		job.RequestedBy,
		job.ChatID,
		string(payload),
		job.CreatedAt,
		now,
		job.MaxAttempts,
		job.NextAttemptAt,
	)
	if err != nil {
		return fmt.Errorf("save review gateway job: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) ReserveAndSaveJob(ctx context.Context, job ReviewGatewayJob, now time.Time, ttl time.Duration) (bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(job.DedupeKey) == "" {
		return false, fmt.Errorf("review gateway state store and dedupe key are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = reviewGatewayDefaultMaxAttempts
	}
	if job.NextAttemptAt == "" {
		job.NextAttemptAt = job.CreatedAt
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return false, fmt.Errorf("encode review gateway job: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin atomic review gateway enqueue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(now)
	_, _ = tx.ExecContext(ctx, "DELETE FROM review_gateway_events WHERE expires_at <= ?", nowText)
	reservation, err := tx.ExecContext(
		ctx,
		"INSERT OR IGNORE INTO review_gateway_events(dedupe_key, received_at, expires_at) VALUES(?, ?, ?)",
		job.DedupeKey,
		nowText,
		reviewGatewayTimestamp(now.Add(ttl)),
	)
	if err != nil {
		return false, fmt.Errorf("reserve atomic review gateway event: %w", err)
	}
	affected, err := reservation.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read atomic review gateway reservation: %w", err)
	}
	if affected == 0 {
		var existingJobID string
		err := tx.QueryRowContext(
			ctx,
			"SELECT job_id FROM review_gateway_jobs WHERE dedupe_key=? LIMIT 1",
			job.DedupeKey,
		).Scan(&existingJobID)
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("inspect atomic review gateway reservation: %w", err)
		}
		// Recover a reservation left by the P2.0 two-transaction enqueue path.
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE review_gateway_events SET received_at=?, expires_at=? WHERE dedupe_key=?",
			nowText,
			reviewGatewayTimestamp(now.Add(ttl)),
			job.DedupeKey,
		); err != nil {
			return false, fmt.Errorf("recover orphan review gateway reservation: %w", err)
		}
	}
	saved, err := tx.ExecContext(
		ctx,
		`INSERT INTO review_gateway_jobs(
			job_id, dedupe_key, status, action, repository, pr_number,
			requested_by, chat_id, payload_json, created_at, updated_at,
			error_summary, attempt_count, max_attempts, next_attempt_at
		) VALUES(?, ?, 'queued', ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?)
		ON CONFLICT(job_id) DO NOTHING`,
		job.JobID,
		job.DedupeKey,
		job.Action,
		job.Repository,
		job.PRNumber,
		job.RequestedBy,
		job.ChatID,
		string(payload),
		job.CreatedAt,
		nowText,
		job.MaxAttempts,
		job.NextAttemptAt,
	)
	if err != nil {
		return false, fmt.Errorf("save atomic review gateway job: %w", err)
	}
	savedCount, err := saved.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read atomic review gateway job insert: %w", err)
	}
	if savedCount != 1 {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit atomic review gateway enqueue: %w", err)
	}
	return true, nil
}

func (s *SQLiteReviewGatewayStore) ClaimReadyJobs(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayJob, error) {
	normalizeReviewGatewayClaimOptions(&opts)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin review gateway job claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(opts.Now)
	_, err = tx.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status='queued', lease_owner='', lease_expires_at='', updated_at=?
		 WHERE status='running' AND lease_expires_at != '' AND lease_expires_at <= ?`,
		nowText,
		nowText,
	)
	if err != nil {
		return nil, fmt.Errorf("recover expired review gateway jobs: %w", err)
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT job_id, payload_json, attempt_count, max_attempts
		 FROM review_gateway_jobs
		 WHERE status='queued'
		   AND (?='' OR queue_class=? )
		   AND (next_attempt_at='' OR next_attempt_at <= ?)
		   AND attempt_count < max_attempts
		 ORDER BY priority, created_at, job_id
		 LIMIT ?`,
		opts.QueueClass,
		opts.QueueClass,
		nowText,
		opts.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ready review gateway jobs: %w", err)
	}
	type candidate struct {
		id          string
		payload     string
		attempts    int
		maxAttempts int
	}
	candidates := []candidate{}
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.payload, &item.attempts, &item.maxAttempts); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan ready review gateway job: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close ready review gateway jobs: %w", err)
	}
	result := []ReviewGatewayJob{}
	for _, candidate := range candidates {
		leaseExpiresAt := reviewGatewayTimestamp(opts.Now.Add(opts.LeaseDuration))
		update, err := tx.ExecContext(
			ctx,
			`UPDATE review_gateway_jobs
			 SET status='running',
			     attempt_count=attempt_count+1,
			     lease_owner=?,
			     lease_expires_at=?,
			     updated_at=?
			 WHERE job_id=? AND status='queued'`,
			opts.LeaseOwner,
			leaseExpiresAt,
			nowText,
			candidate.id,
		)
		if err != nil {
			return nil, fmt.Errorf("lease review gateway job %s: %w", candidate.id, err)
		}
		affected, err := update.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("read review gateway job lease %s: %w", candidate.id, err)
		}
		if affected != 1 {
			continue
		}
		var job ReviewGatewayJob
		if err := json.Unmarshal([]byte(candidate.payload), &job); err != nil {
			return nil, fmt.Errorf("decode review gateway job %s: %w", candidate.id, err)
		}
		job.Status = "running"
		job.AttemptCount = candidate.attempts + 1
		job.MaxAttempts = candidate.maxAttempts
		job.LeaseOwner = opts.LeaseOwner
		job.LeaseExpiresAt = leaseExpiresAt
		result = append(result, job)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit review gateway job claims: %w", err)
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) CompleteJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error {
	return s.completeJob(ctx, job, result, true)
}

// CompleteJobForOperationPlanning persists the result without exposing it to
// the legacy reply dispatcher. The durable Operation planner becomes the sole
// production owner of Card, Reply, Base, Doc and Task side effects.
func (s *SQLiteReviewGatewayStore) CompleteJobForOperationPlanning(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult) error {
	return s.completeJob(ctx, job, result, false)
}

func (s *SQLiteReviewGatewayStore) completeJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, legacyReply bool) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode review gateway result: %w", err)
	}
	replyStatus := "none"
	replyNext := ""
	planStatus := "none"
	if !legacyReply {
		planStatus = "pending"
	}
	completedAt := reviewGatewayResultCompletionTime(result)
	if legacyReply && shouldDispatchReviewGatewayResult(job) {
		replyStatus = "pending"
		replyNext = reviewGatewayTimestamp(completedAt)
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status='completed',
		     result_json=?,
		     error_summary='',
		     lease_owner='',
		     lease_expires_at='',
		     reply_status=?,
		     reply_next_attempt_at=?,
		     operation_plan_status=?,
		     operation_plan_next_attempt_at=?,
		     updated_at=?
		 WHERE job_id=? AND status='running' AND lease_owner=?`,
		string(resultJSON),
		replyStatus,
		replyNext,
		planStatus,
		reviewGatewayTimestamp(completedAt),
		reviewGatewayTimestamp(completedAt),
		job.JobID,
		job.LeaseOwner,
	)
	return requireReviewGatewayJobUpdate(update, err, job.JobID, "complete")
}

func reviewGatewayResultCompletionTime(result ReviewGatewayExecutionResult) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(result.CompletedAt)); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

func (s *SQLiteReviewGatewayStore) RetryOrFailJob(ctx context.Context, job ReviewGatewayJob, result ReviewGatewayExecutionResult, errorSummary string, now time.Time) (bool, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return false, fmt.Errorf("encode failed review gateway result: %w", err)
	}
	willRetry := job.AttemptCount < job.MaxAttempts
	status := "failed"
	nextAttempt := ""
	replyStatus := "none"
	replyNext := ""
	if willRetry {
		status = "queued"
		nextAttempt = reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(job.AttemptCount)))
	} else if shouldDispatchReviewGatewayResult(job) {
		replyStatus = "pending"
		replyNext = reviewGatewayTimestamp(now)
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status=?,
		     next_attempt_at=?,
		     lease_owner='',
		     lease_expires_at='',
		     result_json=?,
		     error_summary=?,
		     reply_status=?,
		     reply_next_attempt_at=?,
		     updated_at=?
		 WHERE job_id=? AND status='running' AND lease_owner=?`,
		status,
		nextAttempt,
		string(resultJSON),
		redactReviewGatewayError(errorSummary),
		replyStatus,
		replyNext,
		reviewGatewayTimestamp(now),
		job.JobID,
		job.LeaseOwner,
	)
	if err := requireReviewGatewayJobUpdate(update, err, job.JobID, "retry or fail"); err != nil {
		return false, err
	}
	return willRetry, nil
}

func (s *SQLiteReviewGatewayStore) RecordHandlerLatency(ctx context.Context, jobID string, latencyMs int64) error {
	if strings.TrimSpace(jobID) == "" {
		return nil
	}
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE review_gateway_jobs SET handler_latency_ms=? WHERE job_id=?",
		latencyMs,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("record review gateway handler latency: %w", err)
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) ClaimPendingReplies(ctx context.Context, opts ReviewGatewayClaimOptions) ([]ReviewGatewayPendingReply, error) {
	normalizeReviewGatewayClaimOptions(&opts)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin review gateway reply claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	nowText := reviewGatewayTimestamp(opts.Now)
	_, err = tx.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status='pending', reply_lease_owner='', reply_lease_expires_at='', updated_at=?
		 WHERE reply_status='sending'
		   AND reply_lease_expires_at != ''
		   AND reply_lease_expires_at <= ?`,
		nowText,
		nowText,
	)
	if err != nil {
		return nil, fmt.Errorf("recover expired review gateway replies: %w", err)
	}
	rows, err := tx.QueryContext(
		ctx,
		`SELECT job_id, payload_json, result_json, reply_attempt_count
		 FROM review_gateway_jobs
		 WHERE reply_status='pending'
		   AND result_json != ''
		   AND (reply_next_attempt_at='' OR reply_next_attempt_at <= ?)
		   AND reply_attempt_count < ?
		 ORDER BY updated_at, job_id
		 LIMIT ?`,
		nowText,
		reviewGatewayReplyMaxAttempts,
		opts.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending review gateway replies: %w", err)
	}
	type replyCandidate struct {
		id         string
		jobJSON    string
		resultJSON string
		attempts   int
	}
	candidates := []replyCandidate{}
	for rows.Next() {
		var candidate replyCandidate
		if err := rows.Scan(&candidate.id, &candidate.jobJSON, &candidate.resultJSON, &candidate.attempts); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan pending review gateway reply: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close pending review gateway replies: %w", err)
	}
	result := []ReviewGatewayPendingReply{}
	for _, candidate := range candidates {
		leaseExpiresAt := reviewGatewayTimestamp(opts.Now.Add(opts.LeaseDuration))
		update, err := tx.ExecContext(
			ctx,
			`UPDATE review_gateway_jobs
			 SET reply_status='sending',
			     reply_attempt_count=reply_attempt_count+1,
			     reply_lease_owner=?,
			     reply_lease_expires_at=?,
			     updated_at=?
			 WHERE job_id=? AND reply_status='pending'`,
			opts.LeaseOwner,
			leaseExpiresAt,
			nowText,
			candidate.id,
		)
		if err != nil {
			return nil, fmt.Errorf("lease review gateway reply %s: %w", candidate.id, err)
		}
		affected, err := update.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("read review gateway reply lease %s: %w", candidate.id, err)
		}
		if affected != 1 {
			continue
		}
		var job ReviewGatewayJob
		var executionResult ReviewGatewayExecutionResult
		if err := json.Unmarshal([]byte(candidate.jobJSON), &job); err != nil {
			return nil, fmt.Errorf("decode review gateway reply job %s: %w", candidate.id, err)
		}
		if err := json.Unmarshal([]byte(candidate.resultJSON), &executionResult); err != nil {
			return nil, fmt.Errorf("decode review gateway result %s: %w", candidate.id, err)
		}
		result = append(result, ReviewGatewayPendingReply{
			Job:          job,
			Result:       executionResult,
			AttemptCount: candidate.attempts + 1,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit review gateway reply claims: %w", err)
	}
	return result, nil
}

func (s *SQLiteReviewGatewayStore) MarkReplySent(ctx context.Context, jobID, messageID string, now time.Time) error {
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status='sent',
		     reply_message_id=?,
		     reply_error_summary='',
		     reply_requires_reconciliation=0,
		     reply_lease_owner='',
		     reply_lease_expires_at='',
		     updated_at=?
		 WHERE job_id=? AND reply_status='sending'`,
		messageID,
		reviewGatewayTimestamp(now),
		jobID,
	)
	return requireReviewGatewayJobUpdate(update, err, jobID, "mark reply sent")
}

func (s *SQLiteReviewGatewayStore) MarkReplyUnknown(
	ctx context.Context,
	jobID,
	messageID,
	errorSummary string,
	now time.Time,
) error {
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status='unknown',
		     reply_message_id=?,
		     reply_error_summary=?,
		     reply_requires_reconciliation=1,
		     reply_lease_owner='',
		     reply_lease_expires_at='',
		     updated_at=?
		 WHERE job_id=? AND reply_status='sending'`,
		messageID,
		redactReviewGatewayError(errorSummary),
		reviewGatewayTimestamp(now),
		jobID,
	)
	return requireReviewGatewayJobUpdate(update, err, jobID, "mark reply unknown")
}

func (s *SQLiteReviewGatewayStore) MarkReplyFailed(ctx context.Context, pending ReviewGatewayPendingReply, errorSummary string, now time.Time) (bool, error) {
	willRetry := pending.AttemptCount < reviewGatewayReplyMaxAttempts
	status := "failed"
	nextAttempt := ""
	if willRetry {
		status = "pending"
		nextAttempt = reviewGatewayTimestamp(now.Add(reviewGatewayRetryDelay(pending.AttemptCount)))
	}
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET reply_status=?,
		     reply_next_attempt_at=?,
		     reply_error_summary=?,
		     reply_requires_reconciliation=0,
		     reply_lease_owner='',
		     reply_lease_expires_at='',
		     updated_at=?
		 WHERE job_id=? AND reply_status='sending'`,
		status,
		nextAttempt,
		redactReviewGatewayError(errorSummary),
		reviewGatewayTimestamp(now),
		pending.Job.JobID,
	)
	if err := requireReviewGatewayJobUpdate(update, err, pending.Job.JobID, "mark reply failed"); err != nil {
		return false, err
	}
	return willRetry, nil
}

func (s *SQLiteReviewGatewayStore) UpdateJobStatus(ctx context.Context, jobID, status, errorSummary string) error {
	update, err := s.db.ExecContext(
		ctx,
		`UPDATE review_gateway_jobs
		 SET status=?, updated_at=?, error_summary=?,
		     lease_owner='', lease_expires_at=''
		 WHERE job_id=?`,
		status,
		reviewGatewayTimestamp(time.Now()),
		redactReviewGatewayError(errorSummary),
		jobID,
	)
	return requireReviewGatewayJobUpdate(update, err, jobID, "update status")
}

func (s *SQLiteReviewGatewayStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func NewReviewGatewayQueue(gateway *ReviewGateway, store ReviewGatewayJobStore, capacity int, onResult func(ReviewGatewayJobOutcome)) *ReviewGatewayQueue {
	if capacity <= 0 {
		capacity = 128
	}
	if store == nil {
		store = NewMemoryReviewGatewayJobStore()
	}
	return &ReviewGatewayQueue{
		gateway:       gateway,
		store:         store,
		wake:          make(chan struct{}, 1),
		latencies:     make(chan reviewGatewayLatencyObservation, capacity),
		leaseOwner:    fmt.Sprintf("worker-%d-%d", os.Getpid(), time.Now().UnixNano()),
		leaseDuration: 2 * time.Minute,
		pollInterval:  time.Second,
		onResult:      onResult,
		now:           time.Now,
		limits:        DefaultReviewQueueLimits(),
	}
}

func (q *ReviewGatewayQueue) ConfigureAdmission(limits ReviewQueueLimits) {
	if q != nil {
		q.limits = limits.normalized()
	}
}

func (q *ReviewGatewayQueue) UseOperationOutbox(planner *ReviewOperationPlanner, wake func()) {
	if q == nil {
		return
	}
	q.operationPlanner = planner
	q.operationWake = wake
	if planner != nil {
		planner.OnPlanned = func([]ReviewOperation) {
			if wake != nil {
				wake()
			}
		}
	}
}

func (q *ReviewGatewayQueue) Enqueue(ctx context.Context, event ReviewGatewayEvent) ReviewGatewayReceipt {
	if q == nil || q.gateway == nil {
		return ReviewGatewayReceipt{
			SchemaVersion: reviewGatewaySchemaVersion,
			Mode:          "preview",
			Reason:        "gateway_not_configured",
		}
	}
	if sqliteStore, ok := q.store.(*SQLiteReviewGatewayStore); ok {
		if dedupeStore, sameStore := q.gateway.deduper.(*SQLiteReviewGatewayStore); sameStore && dedupeStore == sqliteStore {
			receipt, err := q.gateway.planWithoutReserveContext(ctx, event)
			if err != nil {
				receipt.Accepted = false
				receipt.Reason = "state_store_failed"
				return receipt
			}
			if !receipt.Accepted || receipt.Job == nil {
				return receipt
			}
			admission, err := sqliteStore.AdmitReviewGatewayJob(ctx, *receipt.Job, "user_command", q.now().UTC(), q.limits)
			if err != nil {
				receipt.Accepted = false
				receipt.Reason = reviewQueueAdmissionReason(err)
				receipt.Job = nil
				return receipt
			}
			if admission.Duplicate {
				receipt.Accepted = false
				receipt.Duplicate = true
				receipt.Reason = "duplicate_event"
				receipt.Job = nil
				return receipt
			}
			receipt.Job.JobID = admission.JobID
			q.Wake()
			return receipt
		}
	}
	receipt, err := q.gateway.PlanContext(ctx, event)
	if err != nil {
		receipt.Accepted = false
		receipt.Reason = "state_store_failed"
		return receipt
	}
	if !receipt.Accepted || receipt.Job == nil {
		return receipt
	}
	job := *receipt.Job
	if err := q.store.SaveJob(ctx, job); err != nil {
		q.gateway.Release(receipt)
		receipt.Accepted = false
		receipt.Reason = "state_store_failed"
		receipt.Job = nil
		return receipt
	}
	q.Wake()
	return receipt
}

func (q *ReviewGatewayQueue) EnqueuePreparedJob(ctx context.Context, job ReviewGatewayJob) (bool, error) {
	result, err := q.EnqueuePreparedJobWithAdmission(ctx, job, normalizeReviewConsumerSourceType("", job))
	return result.Created, err
}

func (q *ReviewGatewayQueue) EnqueuePreparedJobWithAdmission(ctx context.Context, job ReviewGatewayJob, sourceType string) (ReviewJobAdmissionResult, error) {
	if q == nil || q.gateway == nil || q.store == nil {
		return ReviewJobAdmissionResult{}, fmt.Errorf("review gateway queue is not configured")
	}
	if strings.TrimSpace(job.JobID) == "" || strings.TrimSpace(job.DedupeKey) == "" {
		return ReviewJobAdmissionResult{}, fmt.Errorf("prepared review gateway job identity is required")
	}
	now := q.now().UTC()
	if sqliteStore, ok := q.store.(*SQLiteReviewGatewayStore); ok {
		if dedupeStore, sameStore := q.gateway.deduper.(*SQLiteReviewGatewayStore); sameStore && dedupeStore == sqliteStore {
			result, err := sqliteStore.AdmitReviewGatewayJob(ctx, job, sourceType, now, q.limits)
			if err != nil || result.Duplicate {
				return result, err
			}
			q.Wake()
			return result, nil
		}
	}
	reserved, err := q.gateway.deduper.Reserve(job.DedupeKey, now, 24*time.Hour)
	if err != nil || !reserved {
		return ReviewJobAdmissionResult{Created: reserved, Duplicate: !reserved, JobID: job.JobID}, err
	}
	if err := q.store.SaveJob(ctx, job); err != nil {
		q.gateway.deduper.Release(job.DedupeKey)
		return ReviewJobAdmissionResult{}, err
	}
	q.Wake()
	return ReviewJobAdmissionResult{Created: true, JobID: job.JobID}, nil
}

func reviewQueueAdmissionReason(err error) string {
	var admission *ReviewQueueAdmissionError
	if errors.As(err, &admission) {
		switch admission.Code {
		case "global_job_capacity", "chat_job_capacity", "user_job_capacity", "global_operation_capacity", "chat_operation_capacity":
			return "review_queue_busy"
		case "user_rate_limited", "chat_rate_limited":
			return "review_rate_limited"
		default:
			return admission.Code
		}
	}
	return "state_store_failed"
}

func (q *ReviewGatewayQueue) Wake() {
	if q == nil {
		return
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *ReviewGatewayQueue) ObserveHandlerLatency(jobID string, latencyMs int64) {
	if q == nil || strings.TrimSpace(jobID) == "" {
		return
	}
	select {
	case q.latencies <- reviewGatewayLatencyObservation{JobID: jobID, LatencyMs: latencyMs}:
	default:
	}
}

func (q *ReviewGatewayQueue) Run(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) {
	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()
	go q.runLatencyWriter(ctx)
	q.Wake()
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
			q.runReadyJobs(ctx, handler)
		case <-ticker.C:
			q.runReadyJobs(ctx, handler)
		}
	}
}

func (q *ReviewGatewayQueue) runLatencyWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case observation := <-q.latencies:
			metricCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			_ = q.store.RecordHandlerLatency(metricCtx, observation.JobID, observation.LatencyMs)
			cancel()
		}
	}
}

func (q *ReviewGatewayQueue) runReadyJobs(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error)) {
	q.runReadyJobsForClass(ctx, handler, "", q.leaseOwner)
}

func (q *ReviewGatewayQueue) runReadyJobsForClass(ctx context.Context, handler func(context.Context, ReviewGatewayJob) (ReviewGatewayExecutionResult, error), queueClass, leaseOwner string) {
	for {
		claimCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		jobs, err := q.store.ClaimReadyJobs(claimCtx, ReviewGatewayClaimOptions{
			LeaseOwner:    leaseOwner,
			Now:           q.now().UTC(),
			LeaseDuration: q.leaseDuration,
			Limit:         1,
			QueueClass:    queueClass,
		})
		cancel()
		if err != nil || len(jobs) == 0 {
			return
		}
		job := jobs[0]
		var result ReviewGatewayExecutionResult
		var executeErr error
		if handler == nil {
			executeErr = fmt.Errorf("review gateway job handler is not configured")
			result = failedReviewGatewayResult(job, executeErr, q.now())
		} else {
			result, executeErr = handler(ctx, job)
		}
		outcome := ReviewGatewayJobOutcome{Job: job, Result: result, Err: executeErr}
		persistCtx, persistCancel := context.WithTimeout(ctx, 2*time.Second)
		if executeErr == nil {
			if q.operationPlanner != nil {
				if planningStore, ok := q.store.(interface {
					CompleteJobForOperationPlanning(context.Context, ReviewGatewayJob, ReviewGatewayExecutionResult) error
				}); ok {
					executeErr = planningStore.CompleteJobForOperationPlanning(persistCtx, job, result)
				} else {
					executeErr = fmt.Errorf("review operation planning requires a compatible durable job store")
				}
			} else {
				executeErr = q.store.CompleteJob(persistCtx, job, result)
			}
			outcome.Err = executeErr
		} else {
			outcome.WillRetry, err = q.store.RetryOrFailJob(
				persistCtx,
				job,
				result,
				executeErr.Error(),
				q.now().UTC(),
			)
			if err != nil {
				outcome.Err = err
			}
		}
		persistCancel()
		if executeErr == nil && q.operationPlanner != nil {
			q.operationPlanner.Wake()
		}
		if q.onResult != nil {
			q.onResult(outcome)
		}
		if outcome.WillRetry {
			return
		}
	}
}

func failedReviewGatewayResult(job ReviewGatewayJob, err error, now time.Time) ReviewGatewayExecutionResult {
	return ReviewGatewayExecutionResult{
		SchemaVersion:   reviewGatewayResultSchema,
		JobID:           job.JobID,
		Status:          "failed",
		Mode:            "preview",
		Action:          job.Action,
		Repository:      job.Repository,
		PRNumber:        job.PRNumber,
		RequestedBy:     job.RequestedBy,
		ReadOnlyGitLink: true,
		MutatesGitLink:  false,
		CompletedAt:     now.UTC().Format(time.RFC3339),
		Error:           redactReviewGatewayError(err.Error()),
		AttemptCount:    job.AttemptCount,
	}
}

func normalizeReviewGatewayClaimOptions(opts *ReviewGatewayClaimOptions) {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 2 * time.Minute
	}
	if opts.Limit <= 0 {
		opts.Limit = 1
	}
	if strings.TrimSpace(opts.LeaseOwner) == "" {
		opts.LeaseOwner = "review-gateway-worker"
	}
}

func reviewGatewayRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<(attempt-1)) * 2 * time.Second
}

func reviewGatewayTimeDue(value string, now time.Time) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err != nil || !parsed.After(now)
}

func reviewGatewayTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func requireReviewGatewayJobUpdate(result sql.Result, err error, jobID, action string) error {
	if err != nil {
		return fmt.Errorf("%s review gateway job %s: %w", action, jobID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read %s review gateway job %s: %w", action, jobID, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s review gateway job %q: lease or status changed", action, jobID)
	}
	return nil
}

func redactReviewGatewayError(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
	value = reviewGatewayErrorURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return "https://..."
		}
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.Fragment = ""
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i := range parts {
			if i > 0 && (strings.EqualFold(parts[i-1], "hook") || strings.EqualFold(parts[i-1], "webhook")) {
				parts[i] = "***"
			}
		}
		parsed.Path = "/" + strings.Join(parts, "/")
		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			parsed.Path = ""
		}
		return parsed.String()
	})
	for _, rule := range reviewGatewayErrorRules {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	for _, name := range []string{
		"GITLINK_TOKEN",
		"GITLINK_ACCESS_TOKEN",
		"FEISHU_APP_SECRET",
		"FEISHU_WEBHOOK_SECRET",
		"FEISHU_WEBHOOK_URL",
	} {
		if secret := strings.TrimSpace(os.Getenv(name)); len(secret) >= 6 {
			value = strings.ReplaceAll(value, secret, "***")
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 512 {
		value = string(runes[:512]) + "..."
	}
	return value
}
