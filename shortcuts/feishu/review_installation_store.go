package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SyncReviewGatewayConfiguration atomically replaces the local runtime view of
// GitLink installations and chat bindings. It persists credential references,
// never credential values. The validated bindings file remains the source of
// truth; Base or chat-based administration must produce the same schema before
// it can be applied here.
func (s *SQLiteReviewGatewayStore) SyncReviewGatewayConfiguration(
	ctx context.Context,
	bindings ReviewGatewayBindings,
	source string,
	now time.Time,
) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("review gateway state store is required")
	}
	normalized, err := normalizeReviewGatewayBindings(bindings)
	if err != nil {
		return fmt.Errorf("validate review gateway configuration: %w", err)
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("encode review gateway configuration: %w", err)
	}
	digest := sha256.Sum256(canonical)
	fingerprint := hex.EncodeToString(digest[:])
	appliedAt := reviewGatewayTimestamp(now)
	source = strings.TrimSpace(source)
	if source == "" {
		source = "unknown"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin review gateway configuration sync: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		"DELETE FROM review_identity_bindings",
		"DELETE FROM chat_repository_bindings",
		"DELETE FROM installation_repositories",
		"DELETE FROM gitlink_installations",
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("clear review gateway configuration snapshot: %w", err)
		}
	}

	repositoryCount := 0
	for _, installation := range normalized.Installations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO gitlink_installations(
			installation_id, gitlink_host, owner, credential_ref, operation_mode,
			allow_public_read, webhook_id, webhook_secret_ref, enabled, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			installation.InstallationID,
			installation.GitLinkHost,
			installation.Owner,
			installation.CredentialRef,
			installation.OperationMode,
			boolToSQLiteInteger(installation.AllowPublicRead),
			installation.WebhookID,
			installation.WebhookSecretRef,
			boolToSQLiteInteger(installation.Enabled),
			appliedAt,
		); err != nil {
			return fmt.Errorf("store GitLink installation %q: %w", installation.InstallationID, err)
		}
		for _, repository := range installation.AllowedRepositories {
			if _, err := tx.ExecContext(ctx, `INSERT INTO installation_repositories(
				installation_id, repository, updated_at
			) VALUES(?, ?, ?)`, installation.InstallationID, repository, appliedAt); err != nil {
				return fmt.Errorf("store GitLink installation repository %q: %w", repository, err)
			}
			repositoryCount++
		}
	}

	for _, binding := range normalized.Bindings {
		adminJSON, err := json.Marshal(binding.AdminUserIDs)
		if err != nil {
			return fmt.Errorf("encode chat binding administrators: %w", err)
		}
		allowedJSON, err := json.Marshal(binding.AllowedUserIDs)
		if err != nil {
			return fmt.Errorf("encode chat binding users: %w", err)
		}
		for _, repository := range binding.Repositories {
			if _, err := tx.ExecContext(ctx, `INSERT INTO chat_repository_bindings(
				chat_id, installation_id, repository, is_default, enabled,
				allow_public_read, admin_user_ids_json, allowed_user_ids_json, updated_at
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				binding.ChatID,
				binding.InstallationID,
				repository,
				boolToSQLiteInteger(repository == binding.DefaultRepository),
				boolToSQLiteInteger(binding.Enabled),
				boolToSQLiteInteger(binding.AllowPublicRead),
				string(adminJSON),
				string(allowedJSON),
				appliedAt,
			); err != nil {
				return fmt.Errorf("store chat repository binding %q/%q: %w", binding.ChatID, repository, err)
			}
		}
	}
	for _, identity := range normalized.IdentityBindings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO review_identity_bindings(
			installation_id, feishu_user_id, gitlink_login, verification_method,
			verified_at, enabled, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			identity.InstallationID,
			identity.FeishuUserID,
			identity.GitLinkLogin,
			identity.VerificationMethod,
			identity.VerifiedAt,
			boolToSQLiteInteger(identity.Enabled),
			appliedAt,
		); err != nil {
			return fmt.Errorf("store Review identity binding for %q: %w", identity.FeishuUserID, err)
		}
	}

	auditID := fmt.Sprintf("config-%d-%s", now.UTC().UnixNano(), fingerprint[:12])
	if _, err := tx.ExecContext(ctx, `INSERT INTO review_gateway_configuration_audit(
		audit_id, config_fingerprint, source, installation_count, binding_count,
		repository_count, applied_at
	) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		auditID,
		fingerprint,
		source,
		len(normalized.Installations),
		len(normalized.Bindings),
		repositoryCount,
		appliedAt,
	); err != nil {
		return fmt.Errorf("audit review gateway configuration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit review gateway configuration sync: %w", err)
	}
	return nil
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
