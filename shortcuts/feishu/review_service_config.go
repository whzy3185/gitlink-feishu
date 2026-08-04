package feishu

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type ReviewServiceConfig struct {
	StateDB                 string        `json:"-"`
	AppID                   string        `json:"-"`
	AdminListen             string        `json:"admin_listen"`
	WebhookListen           string        `json:"webhook_listen,omitempty"`
	AdminTokenRef           string        `json:"admin_token_ref,omitempty"`
	ShutdownTimeout         time.Duration `json:"shutdown_timeout"`
	ReadDrainTimeout        time.Duration `json:"read_drain_timeout"`
	OperationDrainTimeout   time.Duration `json:"operation_drain_timeout"`
	WorkerHeartbeatInterval time.Duration `json:"worker_heartbeat_interval"`
	WorkerStaleAfter        time.Duration `json:"worker_stale_after"`
	MetricsEnabled          bool          `json:"metrics_enabled"`
	BackupDirectory         string        `json:"-"`
	BackupInterval          time.Duration `json:"backup_interval"`
	BackupRetentionCount    int           `json:"backup_retention_count"`
	RetentionInterval       time.Duration `json:"retention_interval"`
	WALCheckpointInterval   time.Duration `json:"wal_checkpoint_interval"`
	InstanceHeartbeat       time.Duration `json:"instance_heartbeat_interval"`
	ServiceVersion          string        `json:"service_version,omitempty"`
	CommitSHA               string        `json:"commit_sha,omitempty"`
}

func DefaultReviewServiceConfig() ReviewServiceConfig {
	return ReviewServiceConfig{
		AdminListen:             "127.0.0.1:8787",
		ShutdownTimeout:         30 * time.Second,
		ReadDrainTimeout:        20 * time.Second,
		OperationDrainTimeout:   25 * time.Second,
		WorkerHeartbeatInterval: 5 * time.Second,
		WorkerStaleAfter:        20 * time.Second,
		MetricsEnabled:          true,
		BackupRetentionCount:    7,
		WALCheckpointInterval:   6 * time.Hour,
		InstanceHeartbeat:       10 * time.Second,
	}
}

func (c ReviewServiceConfig) normalized() ReviewServiceConfig {
	d := DefaultReviewServiceConfig()
	if strings.TrimSpace(c.AdminListen) == "" {
		c.AdminListen = d.AdminListen
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = d.ShutdownTimeout
	}
	if c.ReadDrainTimeout == 0 {
		c.ReadDrainTimeout = d.ReadDrainTimeout
	}
	if c.OperationDrainTimeout == 0 {
		c.OperationDrainTimeout = d.OperationDrainTimeout
	}
	if c.WorkerHeartbeatInterval == 0 {
		c.WorkerHeartbeatInterval = d.WorkerHeartbeatInterval
	}
	if c.WorkerStaleAfter == 0 {
		c.WorkerStaleAfter = d.WorkerStaleAfter
	}
	if c.BackupRetentionCount == 0 {
		c.BackupRetentionCount = d.BackupRetentionCount
	}
	if c.WALCheckpointInterval == 0 {
		c.WALCheckpointInterval = d.WALCheckpointInterval
	}
	if c.InstanceHeartbeat == 0 {
		c.InstanceHeartbeat = d.InstanceHeartbeat
	}
	return c
}

func (c ReviewServiceConfig) Validate() error {
	c = c.normalized()
	if err := validateReviewAdminListenAddress(c.AdminListen); err != nil {
		return err
	}
	if c.ShutdownTimeout < 5*time.Second || c.ShutdownTimeout > 5*time.Minute {
		return fmt.Errorf("review service shutdown timeout must be between 5s and 5m")
	}
	if c.ReadDrainTimeout < time.Second || c.ReadDrainTimeout > 5*time.Minute {
		return fmt.Errorf("review service read drain timeout must be between 1s and 5m")
	}
	if c.OperationDrainTimeout < time.Second || c.OperationDrainTimeout > 5*time.Minute {
		return fmt.Errorf("review service operation drain timeout must be between 1s and 5m")
	}
	if c.WorkerHeartbeatInterval < time.Second || c.WorkerHeartbeatInterval > time.Minute {
		return fmt.Errorf("review service heartbeat interval must be between 1s and 1m")
	}
	if c.WorkerStaleAfter < 2*c.WorkerHeartbeatInterval || c.WorkerStaleAfter > 20*c.WorkerHeartbeatInterval {
		return fmt.Errorf("review service stale-after must be between 2x and 20x heartbeat interval")
	}
	if c.InstanceHeartbeat < time.Second || c.InstanceHeartbeat > time.Minute {
		return fmt.Errorf("review service instance heartbeat must be between 1s and 1m")
	}
	if err := validateOptionalReviewServiceInterval("backup", c.BackupInterval, 15*time.Minute, 7*24*time.Hour); err != nil {
		return err
	}
	if err := validateOptionalReviewServiceInterval("retention", c.RetentionInterval, time.Hour, 7*24*time.Hour); err != nil {
		return err
	}
	if err := validateOptionalReviewServiceInterval("WAL checkpoint", c.WALCheckpointInterval, 5*time.Minute, 24*time.Hour); err != nil {
		return err
	}
	if c.BackupRetentionCount < 1 || c.BackupRetentionCount > 100 {
		return fmt.Errorf("review service backup retention count must be between 1 and 100")
	}
	if c.BackupInterval > 0 && strings.TrimSpace(c.BackupDirectory) == "" {
		return fmt.Errorf("review service backup directory is required when periodic backup is enabled")
	}
	return nil
}

func validateOptionalReviewServiceInterval(name string, value, minimum, maximum time.Duration) error {
	if value == 0 {
		return nil
	}
	if value < minimum || value > maximum {
		return fmt.Errorf("review service %s interval is outside the supported range", name)
	}
	return nil
}

func validateReviewAdminListenAddress(address string) error {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("invalid review admin listen address: %w", err)
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		addresses, lookupErr := net.LookupIP(host)
		if lookupErr != nil || len(addresses) == 0 {
			return fmt.Errorf("resolve review admin localhost: %w", lookupErr)
		}
		for _, address := range addresses {
			if !address.IsLoopback() {
				return fmt.Errorf("review admin listener must resolve only to loopback addresses")
			}
		}
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("review admin listener must bind to a loopback address")
	}
	return nil
}
