package feishu

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const reviewGatewayInstanceSchema = "feishu.review-instance/v1"

type reviewGatewayInstanceMetadata struct {
	SchemaVersion string `json:"schema_version"`
	InstanceID    string `json:"instance_id"`
	PID           int    `json:"pid"`
	Hostname      string `json:"hostname"`
	StateDBHash   string `json:"state_db_hash"`
	AppIDHash     string `json:"app_id_hash"`
	StartedAt     string `json:"started_at"`
}

type reviewGatewayInstanceLock struct {
	path     string
	metadata reviewGatewayInstanceMetadata
	released bool
}

func acquireReviewGatewayInstanceLock(statePath, appID string, now time.Time) (*reviewGatewayInstanceLock, error) {
	absoluteStatePath, err := filepath.Abs(statePath)
	if err != nil {
		return nil, fmt.Errorf("resolve review gateway state path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluteStatePath), 0o700); err != nil {
		return nil, fmt.Errorf("create review gateway state directory: %w", err)
	}
	hostname, _ := os.Hostname()
	metadata := reviewGatewayInstanceMetadata{
		SchemaVersion: reviewGatewayInstanceSchema,
		InstanceID:    newReviewGatewayInstanceID(),
		PID:           os.Getpid(),
		Hostname:      hostname,
		StateDBHash:   reviewGatewayHashIdentifier(absoluteStatePath),
		AppIDHash:     reviewGatewayHashIdentifier(appID),
		StartedAt:     now.UTC().Format(time.RFC3339Nano),
	}
	lockPath := absoluteStatePath + ".lock"
	for attempt := 0; attempt < 2; attempt++ {
		file, openErr := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if openErr == nil {
			encoder := json.NewEncoder(file)
			encoder.SetEscapeHTML(false)
			if encodeErr := encoder.Encode(metadata); encodeErr != nil {
				_ = file.Close()
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("write review gateway instance lock: %w", encodeErr)
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("close review gateway instance lock: %w", closeErr)
			}
			return &reviewGatewayInstanceLock{path: lockPath, metadata: metadata}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			return nil, fmt.Errorf("acquire review gateway instance lock: %w", openErr)
		}
		existing, readErr := readReviewGatewayInstanceMetadata(lockPath)
		if readErr != nil {
			return nil, fmt.Errorf("review gateway lock already exists and is unreadable: %w", readErr)
		}
		if !sameReviewGatewayHost(existing.Hostname, hostname) || reviewGatewayProcessAlive(existing.PID) {
			return nil, fmt.Errorf(
				"another review gateway instance is active (instance=%s pid=%d started_at=%s)",
				existing.InstanceID,
				existing.PID,
				existing.StartedAt,
			)
		}
		if removeErr := os.Remove(lockPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, fmt.Errorf("remove stale review gateway instance lock: %w", removeErr)
		}
	}
	return nil, fmt.Errorf("could not acquire review gateway instance lock")
}

func (l *reviewGatewayInstanceLock) Release() error {
	if l == nil || l.released {
		return nil
	}
	existing, err := readReviewGatewayInstanceMetadata(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.released = true
			return nil
		}
		return err
	}
	if existing.InstanceID != l.metadata.InstanceID {
		return fmt.Errorf("review gateway instance lock ownership changed")
	}
	if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	l.released = true
	return nil
}

func readReviewGatewayInstanceMetadata(path string) (reviewGatewayInstanceMetadata, error) {
	var metadata reviewGatewayInstanceMetadata
	payload, err := os.ReadFile(path)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return metadata, err
	}
	if metadata.SchemaVersion != reviewGatewayInstanceSchema || strings.TrimSpace(metadata.InstanceID) == "" {
		return metadata, fmt.Errorf("unsupported review gateway instance lock")
	}
	return metadata, nil
}

func newReviewGatewayInstanceID() string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
}

func reviewGatewayHashIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:6])
}

func sameReviewGatewayHost(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
