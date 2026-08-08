package feishu

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

const legacyReviewInstallationID = "legacy-default"

type GitLinkInstallation struct {
	InstallationID          string   `json:"installation_id"`
	GitLinkHost             string   `json:"gitlink_host,omitempty"`
	Owner                   string   `json:"owner,omitempty"`
	CredentialRef           string   `json:"credential_ref,omitempty"`
	OperationMode           string   `json:"operation_mode"`
	AllowedRepositories     []string `json:"allowed_repositories"`
	AllowPublicRead         bool     `json:"allow_public_read,omitempty"`
	WebhookID               string   `json:"webhook_id,omitempty"`
	WebhookSecretRef        string   `json:"webhook_secret_ref,omitempty"`
	WebhookSignatureMode    string   `json:"webhook_signature_mode,omitempty"`
	WebhookTimestampMode    string   `json:"webhook_timestamp_mode,omitempty"`
	WebhookMaxSkewSeconds   int      `json:"webhook_max_skew_seconds,omitempty"`
	WebhookDeliveryRequired bool     `json:"webhook_delivery_required,omitempty"`
	Enabled                 bool     `json:"enabled"`
}

func normalizeReviewGatewayBindings(input ReviewGatewayBindings) (ReviewGatewayBindings, error) {
	schema := strings.TrimSpace(input.SchemaVersion)
	if schema == "" || schema == reviewGatewayBindingV1 {
		return upgradeReviewGatewayBindingsV1(input)
	}
	if schema != reviewGatewayBindingSchema {
		return ReviewGatewayBindings{}, fmt.Errorf(
			"unsupported review gateway bindings schema %q, want %q or %q",
			schema,
			reviewGatewayBindingSchema,
			reviewGatewayBindingV1,
		)
	}
	return validateReviewGatewayBindingsV2(input)
}

func upgradeReviewGatewayBindingsV1(input ReviewGatewayBindings) (ReviewGatewayBindings, error) {
	if len(input.Bindings) == 0 {
		return validateReviewGatewayBindingsV2(ReviewGatewayBindings{
			SchemaVersion:    reviewGatewayBindingSchema,
			IdentityBindings: append([]ReviewIdentityBinding(nil), input.IdentityBindings...),
		})
	}
	repositories := []string{}
	identities := append([]ReviewIdentityBinding(nil), input.IdentityBindings...)
	for index := range identities {
		identities[index].InstallationID = legacyReviewInstallationID
		if strings.TrimSpace(identities[index].VerificationMethod) == "" {
			identities[index].VerificationMethod = "legacy_admin_config"
		}
	}
	bindings := make([]ReviewChatBinding, 0, len(input.Bindings))
	for _, binding := range input.Bindings {
		repository := strings.TrimSpace(binding.Repository)
		if repository == "" {
			return ReviewGatewayBindings{}, fmt.Errorf("legacy review gateway binding repository is required")
		}
		repositories = append(repositories, repository)
		binding.InstallationID = legacyReviewInstallationID
		binding.Repositories = []string{repository}
		binding.DefaultRepository = repository
		bindings = append(bindings, binding)
	}
	upgraded := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      legacyReviewInstallationID,
			GitLinkHost:         "https://www.gitlink.org.cn",
			OperationMode:       "collaborate",
			AllowedRepositories: sortedUniqueReviewGatewayStrings(repositories),
			Enabled:             true,
		}},
		Bindings:         bindings,
		IdentityBindings: identities,
	}
	return validateReviewGatewayBindingsV2(upgraded)
}

func validateReviewGatewayBindingsV2(input ReviewGatewayBindings) (ReviewGatewayBindings, error) {
	result := ReviewGatewayBindings{
		SchemaVersion:    reviewGatewayBindingSchema,
		Installations:    make([]GitLinkInstallation, 0, len(input.Installations)),
		Bindings:         make([]ReviewChatBinding, 0, len(input.Bindings)),
		IdentityBindings: []ReviewIdentityBinding{},
	}
	installations := map[string]GitLinkInstallation{}
	for _, installation := range input.Installations {
		installation.InstallationID = strings.TrimSpace(installation.InstallationID)
		var err error
		installation.GitLinkHost, err = normalizeReviewGatewayGitLinkHost(installation.GitLinkHost)
		if err != nil {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"GitLink installation %q host: %w",
				installation.InstallationID,
				err,
			)
		}
		installation.Owner = strings.TrimSpace(installation.Owner)
		installation.CredentialRef = strings.TrimSpace(installation.CredentialRef)
		installation.WebhookID = strings.TrimSpace(installation.WebhookID)
		installation.WebhookSecretRef = strings.TrimSpace(installation.WebhookSecretRef)
		installation.WebhookSignatureMode = strings.ToLower(strings.TrimSpace(installation.WebhookSignatureMode))
		if installation.WebhookSignatureMode == "" {
			installation.WebhookSignatureMode = "body_sha256"
		}
		installation.WebhookTimestampMode = strings.ToLower(strings.TrimSpace(installation.WebhookTimestampMode))
		if installation.WebhookTimestampMode == "" {
			installation.WebhookTimestampMode = "optional"
		}
		if installation.WebhookMaxSkewSeconds == 0 {
			installation.WebhookMaxSkewSeconds = 300
		}
		installation.OperationMode = strings.ToLower(strings.TrimSpace(installation.OperationMode))
		if installation.InstallationID == "" {
			return ReviewGatewayBindings{}, fmt.Errorf("GitLink installation_id is required")
		}
		if _, exists := installations[installation.InstallationID]; exists {
			return ReviewGatewayBindings{}, fmt.Errorf("duplicate GitLink installation %q", installation.InstallationID)
		}
		switch installation.OperationMode {
		case "observe", "collaborate", "write":
		default:
			return ReviewGatewayBindings{}, fmt.Errorf(
				"GitLink installation %q has invalid operation_mode %q",
				installation.InstallationID,
				installation.OperationMode,
			)
		}
		installation.AllowedRepositories = sortedUniqueReviewGatewayStrings(installation.AllowedRepositories)
		if len(installation.AllowedRepositories) == 0 && installation.Enabled {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"enabled GitLink installation %q requires an explicit repository allowlist",
				installation.InstallationID,
			)
		}
		for _, repository := range installation.AllowedRepositories {
			if repository == "*" {
				return ReviewGatewayBindings{}, fmt.Errorf("GitLink repository wildcard is not allowed")
			}
			if !reviewGatewayRepositoryPattern.MatchString(repository) {
				return ReviewGatewayBindings{}, fmt.Errorf("GitLink repository %q must use owner/repo", repository)
			}
			if installation.Owner != "" && !strings.HasPrefix(repository, installation.Owner+"/") {
				return ReviewGatewayBindings{}, fmt.Errorf(
					"GitLink repository %q is outside installation owner %q",
					repository,
					installation.Owner,
				)
			}
		}
		if installation.OperationMode == "write" && installation.CredentialRef == "" {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"write-enabled GitLink installation %q requires credential_ref",
				installation.InstallationID,
			)
		}
		if installation.CredentialRef != "" && !reviewGatewaySecretReferencePattern.MatchString(installation.CredentialRef) {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"GitLink installation %q credential_ref must use env:VARIABLE",
				installation.InstallationID,
			)
		}
		if installation.WebhookSecretRef != "" && !reviewGatewaySecretReferencePattern.MatchString(installation.WebhookSecretRef) {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"GitLink installation %q webhook_secret_ref must use env:VARIABLE",
				installation.InstallationID,
			)
		}
		switch installation.WebhookSignatureMode {
		case "body_sha256", "timestamp_body_sha256":
		default:
			return ReviewGatewayBindings{}, fmt.Errorf("GitLink installation %q has invalid webhook_signature_mode %q", installation.InstallationID, installation.WebhookSignatureMode)
		}
		switch installation.WebhookTimestampMode {
		case "required", "optional", "disabled":
		default:
			return ReviewGatewayBindings{}, fmt.Errorf("GitLink installation %q has invalid webhook_timestamp_mode %q", installation.InstallationID, installation.WebhookTimestampMode)
		}
		if installation.WebhookMaxSkewSeconds < 30 || installation.WebhookMaxSkewSeconds > 600 {
			return ReviewGatewayBindings{}, fmt.Errorf("GitLink installation %q webhook_max_skew_seconds must be between 30 and 600", installation.InstallationID)
		}
		installations[installation.InstallationID] = installation
		result.Installations = append(result.Installations, installation)
	}
	if len(installations) == 0 && len(input.Bindings) > 0 {
		return ReviewGatewayBindings{}, fmt.Errorf("review gateway bindings v2 requires at least one GitLink installation")
	}
	for _, binding := range input.Bindings {
		binding.ChatID = strings.TrimSpace(binding.ChatID)
		binding.InstallationID = strings.TrimSpace(binding.InstallationID)
		if binding.ChatID == "" {
			return ReviewGatewayBindings{}, fmt.Errorf("review gateway binding chat_id is required")
		}
		if binding.InstallationID == "" && len(result.Installations) == 1 {
			binding.InstallationID = result.Installations[0].InstallationID
		}
		installation, exists := installations[binding.InstallationID]
		if !exists {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"review gateway binding for chat %q references unknown installation %q",
				binding.ChatID,
				binding.InstallationID,
			)
		}
		repositories := append([]string(nil), binding.Repositories...)
		if legacy := strings.TrimSpace(binding.Repository); legacy != "" {
			repositories = append(repositories, legacy)
		}
		binding.Repositories = sortedUniqueReviewGatewayStrings(repositories)
		if len(binding.Repositories) == 0 {
			return ReviewGatewayBindings{}, fmt.Errorf("review gateway binding for chat %q requires repositories", binding.ChatID)
		}
		for _, repository := range binding.Repositories {
			if !containsReviewGatewayString(installation.AllowedRepositories, repository) {
				return ReviewGatewayBindings{}, fmt.Errorf(
					"repository %q is outside installation %q allowlist",
					repository,
					binding.InstallationID,
				)
			}
		}
		if binding.AllowPublicRead && !installation.AllowPublicRead {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"chat %q enables public read while installation %q disables it",
				binding.ChatID,
				binding.InstallationID,
			)
		}
		binding.DefaultRepository = strings.TrimSpace(binding.DefaultRepository)
		if binding.DefaultRepository == "" {
			binding.DefaultRepository = strings.TrimSpace(binding.Repository)
		}
		if binding.DefaultRepository == "" && len(binding.Repositories) == 1 {
			binding.DefaultRepository = binding.Repositories[0]
		}
		if binding.DefaultRepository != "" &&
			!containsReviewGatewayString(binding.Repositories, binding.DefaultRepository) {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"default repository %q is not bound to chat %q",
				binding.DefaultRepository,
				binding.ChatID,
			)
		}
		binding.Repository = binding.DefaultRepository
		binding.AdminUserIDs = sortedUniqueReviewGatewayStrings(binding.AdminUserIDs)
		binding.AllowedUserIDs = sortedUniqueReviewGatewayStrings(binding.AllowedUserIDs)
		result.Bindings = append(result.Bindings, binding)
	}
	identityKeys := map[string]bool{}
	for _, identity := range input.IdentityBindings {
		identity.InstallationID = strings.TrimSpace(identity.InstallationID)
		identity.FeishuUserID = strings.TrimSpace(identity.FeishuUserID)
		identity.GitLinkLogin = strings.TrimSpace(identity.GitLinkLogin)
		identity.VerificationMethod = strings.TrimSpace(identity.VerificationMethod)
		identity.VerifiedAt = strings.TrimSpace(identity.VerifiedAt)
		if identity.InstallationID == "" && len(result.Installations) == 1 {
			identity.InstallationID = result.Installations[0].InstallationID
		}
		if !identity.Enabled {
			result.IdentityBindings = append(result.IdentityBindings, identity)
			continue
		}
		if identity.FeishuUserID == "" || identity.GitLinkLogin == "" {
			return ReviewGatewayBindings{}, fmt.Errorf("enabled Review identity bindings require feishu_user_id and gitlink_login")
		}
		if _, exists := installations[identity.InstallationID]; !exists {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"Review identity for Feishu user %q references unknown installation %q",
				identity.FeishuUserID,
				identity.InstallationID,
			)
		}
		if identity.VerificationMethod == "" {
			identity.VerificationMethod = "admin_config"
		}
		key := identity.InstallationID + "\x00" + identity.FeishuUserID
		if identityKeys[key] {
			return ReviewGatewayBindings{}, fmt.Errorf(
				"duplicate Review identity for Feishu user %q in installation %q",
				identity.FeishuUserID,
				identity.InstallationID,
			)
		}
		identityKeys[key] = true
		result.IdentityBindings = append(result.IdentityBindings, identity)
	}
	sort.Slice(result.Installations, func(i, j int) bool {
		return result.Installations[i].InstallationID < result.Installations[j].InstallationID
	})
	sort.Slice(result.Bindings, func(i, j int) bool {
		return result.Bindings[i].ChatID < result.Bindings[j].ChatID
	})
	sort.Slice(result.IdentityBindings, func(i, j int) bool {
		left := result.IdentityBindings[i].InstallationID + "\x00" + result.IdentityBindings[i].FeishuUserID
		right := result.IdentityBindings[j].InstallationID + "\x00" + result.IdentityBindings[j].FeishuUserID
		return left < right
	})
	return result, nil
}

func normalizeReviewGatewayGitLinkHost(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("must be an absolute URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("must not contain userinfo, query, or fragment")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
	case "http":
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return "", fmt.Errorf("HTTP is allowed only for loopback test hosts")
		}
	default:
		return "", fmt.Errorf("scheme must be HTTPS")
	}
	return value, nil
}

func resolveReviewGatewayRepository(
	binding ReviewChatBinding,
	installation GitLinkInstallation,
	action string,
	requested string,
) (string, string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", "repository_qualification_required", false
	}
	if containsReviewGatewayString(binding.Repositories, requested) {
		return requested, "", false
	}
	if binding.AllowPublicRead && installation.AllowPublicRead &&
		reviewGatewayActionAllowsPublicRepository(action) {
		return requested, "", true
	}
	return "", "repository_not_bound", false
}

func reviewGatewayActionAllowsPublicRepository(action string) bool {
	return action == "read_review_context"
}

func reviewGatewayActionRequiresCollaborationAuthorization(action string) bool {
	switch action {
	case "claim_review", "release_review", "set_review_deadline":
		return true
	default:
		return false
	}
}

func reviewGatewayActionRequiresRepository(action string) bool {
	switch action {
	case "read_review_queue", "read_review_context", "refresh_review_context",
		"generate_review_draft", "run_agent_review", "claim_review", "release_review",
		"set_review_deadline", "prepare_common_review", "prepare_review_approve",
		"prepare_review_reject", "prepare_reject_close", "prepare_merge", "subscribe_review_events",
		"unsubscribe_review_events",
		"set_review_notification_mode", "set_default_review_repository":
		return true
	default:
		return false
	}
}
