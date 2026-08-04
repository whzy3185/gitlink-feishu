package feishu

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
)

type ReviewServiceInstance struct {
	InstanceID        string                      `json:"instance_id"`
	ProcessID         int                         `json:"process_id"`
	HostnameHash      string                      `json:"hostname_hash"`
	ServiceVersion    string                      `json:"service_version,omitempty"`
	CommitSHA         string                      `json:"commit_sha,omitempty"`
	SchemaVersion     int                         `json:"schema_version"`
	ConfigRevision    int                         `json:"config_revision"`
	ConfigFingerprint string                      `json:"config_fingerprint,omitempty"`
	Status            ReviewServiceState          `json:"status"`
	AdmissionStatus   ReviewServiceAdmissionState `json:"admission_status"`
	StartedAt         string                      `json:"started_at"`
	ReadyAt           string                      `json:"ready_at,omitempty"`
	HeartbeatAt       string                      `json:"heartbeat_at,omitempty"`
	ShutdownStartedAt string                      `json:"shutdown_started_at,omitempty"`
	StoppedAt         string                      `json:"stopped_at,omitempty"`
	LastErrorSummary  string                      `json:"last_error_summary,omitempty"`
}

func (s *SQLiteReviewGatewayStore) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("review service SQLite store is unavailable")
	}
	return s.db.PingContext(ctx)
}

func (s *SQLiteReviewGatewayStore) CurrentSchemaVersion(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("review service SQLite store is unavailable")
	}
	var version sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, err
	}
	return int(version.Int64), nil
}

func latestReviewGatewaySchemaVersion() int {
	if len(reviewGatewaySchemaMigrations) == 0 {
		return 0
	}
	return reviewGatewaySchemaMigrations[len(reviewGatewaySchemaMigrations)-1].Version
}

func (s *SQLiteReviewGatewayStore) SaveReviewServiceInstance(ctx context.Context, instance ReviewServiceInstance) error {
	if strings.TrimSpace(instance.InstanceID) == "" {
		return fmt.Errorf("review service instance ID is required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO review_service_instances(
		instance_id,process_id,hostname_hash,service_version,commit_sha,schema_version,
		config_revision,config_fingerprint,status,admission_status,started_at,ready_at,
		heartbeat_at,shutdown_started_at,stopped_at,last_error_summary
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(instance_id) DO UPDATE SET
		schema_version=excluded.schema_version,config_revision=excluded.config_revision,
		config_fingerprint=excluded.config_fingerprint,status=excluded.status,
		admission_status=excluded.admission_status,ready_at=excluded.ready_at,
		heartbeat_at=excluded.heartbeat_at,shutdown_started_at=excluded.shutdown_started_at,
		stopped_at=excluded.stopped_at,last_error_summary=excluded.last_error_summary`,
		instance.InstanceID, instance.ProcessID, instance.HostnameHash, instance.ServiceVersion,
		instance.CommitSHA, instance.SchemaVersion, instance.ConfigRevision, instance.ConfigFingerprint,
		instance.Status, instance.AdmissionStatus, instance.StartedAt, instance.ReadyAt,
		instance.HeartbeatAt, instance.ShutdownStartedAt, instance.StoppedAt,
		redactReviewGatewayError(instance.LastErrorSummary))
	return err
}

func (s *SQLiteReviewGatewayStore) GetReviewServiceInstance(ctx context.Context, instanceID string) (ReviewServiceInstance, error) {
	var value ReviewServiceInstance
	err := s.db.QueryRowContext(ctx, `SELECT instance_id,process_id,hostname_hash,service_version,
		commit_sha,schema_version,config_revision,config_fingerprint,status,admission_status,
		started_at,ready_at,heartbeat_at,shutdown_started_at,stopped_at,last_error_summary
		FROM review_service_instances WHERE instance_id=?`, strings.TrimSpace(instanceID)).Scan(
		&value.InstanceID, &value.ProcessID, &value.HostnameHash, &value.ServiceVersion,
		&value.CommitSHA, &value.SchemaVersion, &value.ConfigRevision, &value.ConfigFingerprint,
		&value.Status, &value.AdmissionStatus, &value.StartedAt, &value.ReadyAt,
		&value.HeartbeatAt, &value.ShutdownStartedAt, &value.StoppedAt, &value.LastErrorSummary)
	return value, err
}

func newReviewServiceInstance(lock *reviewGatewayInstanceLock, config ReviewServiceConfig, status ReviewServiceStatusSnapshot, schemaVersion, configRevision int, configFingerprint string, now time.Time) ReviewServiceInstance {
	hostname, _ := os.Hostname()
	instanceID := ""
	if lock != nil {
		instanceID = lock.metadata.InstanceID
	}
	return ReviewServiceInstance{
		InstanceID: instanceID, ProcessID: os.Getpid(), HostnameHash: reviewGatewayHashIdentifier(hostname),
		ServiceVersion: config.ServiceVersion, CommitSHA: config.CommitSHA,
		SchemaVersion: schemaVersion, ConfigRevision: configRevision, ConfigFingerprint: configFingerprint,
		Status: status.State, AdmissionStatus: status.Admission, StartedAt: status.StartedAt,
		ReadyAt: status.ReadyAt, HeartbeatAt: reviewGatewayTimestamp(now),
		ShutdownStartedAt: status.ShutdownStartedAt, StoppedAt: status.StoppedAt,
		LastErrorSummary: status.LastErrorSummary,
	}
}

func (l *reviewGatewayInstanceLock) Valid() bool {
	if l == nil || l.released {
		return false
	}
	existing, err := readReviewGatewayInstanceMetadata(l.path)
	return err == nil && existing.InstanceID == l.metadata.InstanceID
}
