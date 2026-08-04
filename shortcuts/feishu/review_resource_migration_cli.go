package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReviewMigrationCommandOptions struct {
	Action          string
	MigrationID     string
	Status          string
	ResourceType    string
	InstallationID  string
	Scope           ReviewResourceScope
	EnableMigration bool
	Method          string
	ChatID          string
	Actor           string
	Reason          string
	Confirmed       bool
}

type reviewMigrationCommandResult struct {
	SchemaVersion string                             `json:"schema_version"`
	Action        string                             `json:"action"`
	ReadOnly      bool                               `json:"read_only"`
	Migrations    []reviewResourceMigrationView      `json:"migrations,omitempty"`
	Migration     *reviewResourceMigrationView       `json:"migration,omitempty"`
	Policies      []reviewResourceScopePolicyView    `json:"policies,omitempty"`
	Audits        []reviewResourceMigrationAuditView `json:"audits,omitempty"`
}

type reviewResourceMigrationView struct {
	MigrationID        string              `json:"migration_id"`
	LegacyWorkItemKey  string              `json:"legacy_work_item_key"`
	TargetWorkItemKey  string              `json:"target_work_item_key,omitempty"`
	ResourceType       string              `json:"resource_type"`
	TargetScope        ReviewResourceScope `json:"target_scope,omitempty"`
	InstallationID     string              `json:"installation_id,omitempty"`
	Repository         string              `json:"repository,omitempty"`
	PRNumber           int                 `json:"pr_number,omitempty"`
	ChatIDHash         string              `json:"chat_id_hash,omitempty"`
	RemoteIDHash       string              `json:"remote_id_hash,omitempty"`
	Status             string              `json:"status"`
	ReasonCode         string              `json:"reason_code,omitempty"`
	VerificationMethod string              `json:"verification_method,omitempty"`
	VerifiedByHash     string              `json:"verified_by_hash,omitempty"`
	VerifiedAt         string              `json:"verified_at,omitempty"`
	AttemptCount       int                 `json:"attempt_count"`
	ErrorSummary       string              `json:"error_summary,omitempty"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

type reviewResourceScopePolicyView struct {
	InstallationID   string              `json:"installation_id"`
	ResourceType     string              `json:"resource_type"`
	TargetScope      ReviewResourceScope `json:"target_scope"`
	MigrationEnabled bool                `json:"migration_enabled"`
	Revision         int                 `json:"revision"`
	UpdatedByHash    string              `json:"updated_by_hash,omitempty"`
	UpdatedAt        string              `json:"updated_at"`
}

type reviewResourceMigrationAuditView struct {
	AuditID     string                 `json:"audit_id"`
	MigrationID string                 `json:"migration_id"`
	Action      string                 `json:"action"`
	FromStatus  string                 `json:"from_status"`
	ToStatus    string                 `json:"to_status"`
	ActorHash   string                 `json:"actor_hash,omitempty"`
	Detail      map[string]interface{} `json:"detail"`
	CreatedAt   string                 `json:"created_at"`
}

func newReviewMigrationShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-migration",
		Description: "Manage fail-closed local Review resource migration plans",
		Long: "Local-only migration administration for legacy Card, Base, Doc, and Task mappings. " +
			"The command never calls GitLink or Feishu APIs and never deletes legacy source records.",
		Flags: []common.Flag{
			{Name: "action", Usage: "scan, list, inspect, policy-list, policy-set, verify, apply, skip, or retry", Required: true},
			{Name: "state-db", Usage: "SQLite Review gateway state database", Default: ".local/review-gateway.db"},
			{Name: "migration-id", Usage: "Migration ID for inspect, verify, apply, skip, or retry"},
			{Name: "status", Usage: "Optional migration status filter"},
			{Name: "resource", Usage: "Resource type or policy resource"},
			{Name: "installation", Usage: "Installation ID or filter"},
			{Name: "scope", Usage: "chat, installation, or disabled"},
			{Name: "enable-migration", Usage: "Allow explicitly scoped legacy migration", Bool: true, Default: "false"},
			{Name: "method", Usage: "Verification method; CLI only accepts operator-confirmed"},
			{Name: "chat-id", Usage: "Expected target Chat ID; stored and printed only as a hash"},
			{Name: "actor", Usage: "Local operator identity; audit stores only its SHA-256 hash"},
			{Name: "reason", Usage: "Required reason for skip"},
			{Name: "yes", Usage: "Explicitly confirm a verify operation", Bool: true, Default: "false"},
		},
		Run: runReviewMigration,
	}
}

func runReviewMigration(runtime *common.RuntimeContext) error {
	store, err := OpenSQLiteReviewGatewayStore(runtime.Arg("state-db"))
	if err != nil {
		return err
	}
	defer store.Close()
	result, err := executeReviewMigrationCommand(context.Background(), store, ReviewMigrationCommandOptions{
		Action:          runtime.Arg("action"),
		MigrationID:     runtime.Arg("migration-id"),
		Status:          runtime.Arg("status"),
		ResourceType:    runtime.Arg("resource"),
		InstallationID:  runtime.Arg("installation"),
		Scope:           ReviewResourceScope(runtime.Arg("scope")),
		EnableMigration: parseBool(runtime.Arg("enable-migration")),
		Method:          runtime.Arg("method"),
		ChatID:          runtime.Arg("chat-id"),
		Actor:           runtime.Arg("actor"),
		Reason:          runtime.Arg("reason"),
		Confirmed:       parseBool(runtime.Arg("yes")),
	}, time.Now().UTC())
	if err != nil {
		return err
	}
	return renderReviewMigrationCommandResult(os.Stdout, result, runtime.Format)
}

func executeReviewMigrationCommand(
	ctx context.Context,
	store *SQLiteReviewGatewayStore,
	opts ReviewMigrationCommandOptions,
	now time.Time,
) (reviewMigrationCommandResult, error) {
	result := reviewMigrationCommandResult{
		SchemaVersion: "feishu.review-resource-migration-command/v1",
		Action:        strings.TrimSpace(opts.Action), ReadOnly: true,
	}
	switch result.Action {
	case "scan":
		result.ReadOnly = false
		if err := store.ScanLegacyReviewResourceMigrations(ctx, now); err != nil {
			return result, err
		}
		plans, err := store.ListReviewResourceMigrations(ctx, opts.Status, opts.ResourceType, opts.InstallationID)
		result.Migrations = reviewResourceMigrationViews(plans)
		return result, err
	case "list":
		plans, err := store.ListReviewResourceMigrations(ctx, opts.Status, opts.ResourceType, opts.InstallationID)
		result.Migrations = reviewResourceMigrationViews(plans)
		return result, err
	case "inspect":
		migration, err := requireReviewMigration(ctx, store, opts.MigrationID)
		if err != nil {
			return result, err
		}
		view := reviewResourceMigrationViewOf(migration)
		result.Migration = &view
		audits, auditErr := store.ListReviewResourceMigrationAudit(ctx, migration.MigrationID)
		result.Audits = reviewResourceMigrationAuditViews(audits)
		return result, auditErr
	case "policy-list":
		policies, err := store.ListReviewResourceScopePolicies(ctx)
		result.Policies = reviewResourceScopePolicyViews(policies)
		return result, err
	case "policy-set":
		result.ReadOnly = false
		if strings.TrimSpace(opts.Actor) == "" {
			return result, fmt.Errorf("policy-set requires --actor")
		}
		policy, err := store.SetReviewResourceScopePolicy(ctx, ReviewResourceScopePolicy{
			InstallationID: opts.InstallationID, ResourceType: opts.ResourceType,
			TargetScope: opts.Scope, MigrationEnabled: opts.EnableMigration, UpdatedBy: opts.Actor,
		}, now)
		if err != nil {
			return result, err
		}
		result.Policies = reviewResourceScopePolicyViews([]ReviewResourceScopePolicy{policy})
		return result, nil
	case "verify":
		result.ReadOnly = false
		method := strings.ReplaceAll(strings.TrimSpace(opts.Method), "-", "_")
		if method != "operator_confirmed" {
			return result, fmt.Errorf("CLI verification requires --method operator-confirmed")
		}
		if strings.TrimSpace(opts.Actor) == "" {
			return result, fmt.Errorf("verify requires --actor")
		}
		migration, err := store.VerifyLegacyReviewResourceMigration(ctx, opts.MigrationID,
			LegacyReviewResourceVerificationInput{
				ExpectedInstallation: opts.InstallationID, ExpectedScope: opts.Scope,
				ExpectedChatID: opts.ChatID, Method: method, Actor: opts.Actor, Confirmed: opts.Confirmed,
			}, OperatorConfirmedLegacyReviewResourceVerifier{}, now)
		if err != nil {
			return result, err
		}
		view := reviewResourceMigrationViewOf(migration)
		result.Migration = &view
		return result, nil
	case "apply":
		result.ReadOnly = false
		if strings.TrimSpace(opts.Actor) == "" {
			return result, fmt.Errorf("apply requires --actor")
		}
		migration, err := store.ApplyLegacyReviewResourceMigration(ctx, opts.MigrationID, opts.Actor, now)
		view := reviewResourceMigrationViewOf(migration)
		result.Migration = &view
		return result, err
	case "skip":
		result.ReadOnly = false
		if strings.TrimSpace(opts.Actor) == "" {
			return result, fmt.Errorf("skip requires --actor")
		}
		migration, err := store.SkipLegacyReviewResourceMigration(ctx, opts.MigrationID, opts.Reason, opts.Actor, now)
		view := reviewResourceMigrationViewOf(migration)
		result.Migration = &view
		return result, err
	case "retry":
		result.ReadOnly = false
		if strings.TrimSpace(opts.Actor) == "" {
			return result, fmt.Errorf("retry requires --actor")
		}
		migration, err := store.RetryLegacyReviewResourceMigration(ctx, opts.MigrationID, opts.Actor, now)
		view := reviewResourceMigrationViewOf(migration)
		result.Migration = &view
		return result, err
	default:
		return result, fmt.Errorf("unsupported review migration action %q", result.Action)
	}
}

func requireReviewMigration(ctx context.Context, store *SQLiteReviewGatewayStore, id string) (ReviewResourceMigration, error) {
	if strings.TrimSpace(id) == "" {
		return ReviewResourceMigration{}, fmt.Errorf("--migration-id is required")
	}
	return store.GetReviewResourceMigration(ctx, id)
}

func reviewResourceMigrationViews(migrations []ReviewResourceMigration) []reviewResourceMigrationView {
	result := make([]reviewResourceMigrationView, 0, len(migrations))
	for _, migration := range migrations {
		result = append(result, reviewResourceMigrationViewOf(migration))
	}
	return result
}

func reviewResourceMigrationViewOf(migration ReviewResourceMigration) reviewResourceMigrationView {
	return reviewResourceMigrationView{
		MigrationID: migration.MigrationID, LegacyWorkItemKey: migration.LegacyWorkItemKey,
		TargetWorkItemKey: migration.TargetWorkItemKey, ResourceType: migration.ResourceType,
		TargetScope: migration.TargetScope, InstallationID: migration.InstallationID,
		Repository: migration.Repository, PRNumber: migration.PRNumber,
		ChatIDHash:   reviewResourceHashPrefix(migration.ChatIDHash),
		RemoteIDHash: reviewResourceHashPrefix(migration.RemoteIDHash),
		Status:       migration.Status, ReasonCode: migration.ReasonCode,
		VerificationMethod: migration.VerificationMethod,
		VerifiedByHash:     reviewResourceHashPrefix(migration.VerifiedByHash),
		VerifiedAt:         migration.VerifiedAt, AttemptCount: migration.AttemptCount,
		ErrorSummary: redactReviewGatewayError(migration.ErrorSummary),
		CreatedAt:    migration.CreatedAt, UpdatedAt: migration.UpdatedAt,
	}
}

func reviewResourceScopePolicyViews(policies []ReviewResourceScopePolicy) []reviewResourceScopePolicyView {
	result := make([]reviewResourceScopePolicyView, 0, len(policies))
	for _, policy := range policies {
		result = append(result, reviewResourceScopePolicyView{
			InstallationID: policy.InstallationID, ResourceType: policy.ResourceType,
			TargetScope: policy.TargetScope, MigrationEnabled: policy.MigrationEnabled,
			Revision: policy.Revision, UpdatedByHash: reviewResourceHashPrefix(reviewResourceIdentifierHash(policy.UpdatedBy)),
			UpdatedAt: policy.UpdatedAt,
		})
	}
	return result
}

func reviewResourceMigrationAuditViews(audits []ReviewResourceMigrationAudit) []reviewResourceMigrationAuditView {
	result := make([]reviewResourceMigrationAuditView, 0, len(audits))
	for _, audit := range audits {
		detail := map[string]interface{}{}
		if err := json.Unmarshal(audit.DetailJSON, &detail); err != nil {
			detail = map[string]interface{}{"error": "invalid audit detail"}
		}
		for key, value := range detail {
			text, ok := value.(string)
			if ok && strings.Contains(strings.ToLower(key), "hash") {
				detail[key] = reviewResourceHashPrefix(text)
			}
		}
		result = append(result, reviewResourceMigrationAuditView{
			AuditID: audit.AuditID, MigrationID: audit.MigrationID, Action: audit.Action,
			FromStatus: audit.FromStatus, ToStatus: audit.ToStatus,
			ActorHash: reviewResourceHashPrefix(audit.ActorHash), Detail: detail, CreatedAt: audit.CreatedAt,
		})
	}
	return result
}

func reviewResourceHashPrefix(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func renderReviewMigrationCommandResult(writer io.Writer, result reviewMigrationCommandResult, format string) error {
	if strings.EqualFold(strings.TrimSpace(format), "table") && len(result.Migrations) > 0 {
		if _, err := fmt.Fprintln(writer, "MIGRATION_ID\tRESOURCE\tSCOPE\tINSTALLATION\tREPOSITORY\tPR\tSTATUS\tREASON"); err != nil {
			return err
		}
		for _, migration := range result.Migrations {
			if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
				migration.MigrationID, migration.ResourceType, migration.TargetScope,
				migration.InstallationID, migration.Repository, migration.PRNumber,
				migration.Status, migration.ReasonCode); err != nil {
				return err
			}
		}
		return nil
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
