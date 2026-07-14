package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type releaseTarget struct {
	Identifier string
	VersionID  string
	View       map[string]interface{}
	Edit       map[string]interface{}
}

func releaseAssetShortcuts(tr *i18n.Translator) []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "assets",
			Description: tr.T("cmd.release.assets.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
			},
			Run: runAssets,
		},
		{
			Name:        "attach",
			Description: tr.T("cmd.release.attach.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
				{Name: "attachment-ids", Usage: "Comma-separated attachment IDs", Required: true},
				{Name: "dry-run", Usage: "Preview the attach request without changing release state", Bool: true, Default: "false"},
			},
			Run: runAttach,
		},
		{
			Name:        "detach",
			Description: tr.T("cmd.release.detach.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
				{Name: "attachment-ids", Usage: "Comma-separated attachment IDs", Required: true},
				{Name: "dry-run", Usage: "Preview the detach request without changing release state", Bool: true, Default: "false"},
			},
			Run: runDetach,
		},
		{
			Name:        "upload",
			Description: tr.T("cmd.release.upload.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
				{Name: "file", Usage: "Local file path", Required: true},
				{Name: "asset-name", Usage: "Override the uploaded asset filename"},
				{Name: "description", Usage: "Attachment description"},
				{Name: "dry-run", Usage: "Preview the upload request without changing release state", Bool: true, Default: "false"},
			},
			Run: runUpload,
		},
	}
}

func runAssets(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	identifier, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}

	view, err := fetchReleaseView(ctx, identifier)
	if err != nil {
		return err
	}
	target := &releaseTarget{
		Identifier: identifier,
		VersionID:  releaseVersionID(identifier, view),
		View:       view,
	}

	result := releaseActionResult(ctx, target, "list_release_assets")
	result["attachment_ids"] = releaseAttachmentIDs(view)
	result["attachments"] = releaseAttachments(view)
	return ctx.OutputData(result)
}

func runAttach(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	identifier, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	requestedIDs, err := parseReleaseAttachmentIDs(ctx.Arg("attachment-ids"))
	if err != nil {
		return err
	}

	target, err := resolveReleaseTarget(ctx, identifier)
	if err != nil {
		return err
	}

	currentIDs := releaseAttachmentIDs(target.Edit)
	nextIDs, addedIDs := mergeReleaseAttachmentIDs(currentIDs, requestedIDs)
	result := releaseActionResult(ctx, target, "attach_release_assets")
	result["dry_run"] = ctx.Arg("dry-run") == "true"
	result["changed"] = len(addedIDs) > 0
	result["current_attachment_ids"] = currentIDs
	result["requested_attachment_ids"] = requestedIDs
	result["added_attachment_ids"] = addedIDs
	result["attachment_ids"] = nextIDs
	if len(addedIDs) == 0 || ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(result)
	}

	env, err := updateReleaseAttachments(ctx, target.VersionID, target.Edit, nextIDs)
	if err != nil {
		return err
	}
	result["release"] = env.Data
	return ctx.OutputData(result)
}

func runDetach(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	identifier, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	requestedIDs, err := parseReleaseAttachmentIDs(ctx.Arg("attachment-ids"))
	if err != nil {
		return err
	}

	target, err := resolveReleaseTarget(ctx, identifier)
	if err != nil {
		return err
	}

	currentIDs := releaseAttachmentIDs(target.Edit)
	nextIDs, removedIDs := removeReleaseAttachmentIDs(currentIDs, requestedIDs)
	result := releaseActionResult(ctx, target, "detach_release_assets")
	result["dry_run"] = ctx.Arg("dry-run") == "true"
	result["changed"] = len(removedIDs) > 0
	result["current_attachment_ids"] = currentIDs
	result["requested_attachment_ids"] = requestedIDs
	result["removed_attachment_ids"] = removedIDs
	result["attachment_ids"] = nextIDs
	if len(removedIDs) == 0 || ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(result)
	}

	env, err := updateReleaseAttachments(ctx, target.VersionID, target.Edit, nextIDs)
	if err != nil {
		return err
	}
	result["release"] = env.Data
	return ctx.OutputData(result)
}

func runUpload(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	identifier, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	filePath, uploadName, size, err := resolveUploadFile(ctx.Arg("file"), ctx.Arg("asset-name"))
	if err != nil {
		return err
	}

	target, err := resolveReleaseTarget(ctx, identifier)
	if err != nil {
		return err
	}

	result := releaseActionResult(ctx, target, "upload_release_asset")
	result["dry_run"] = ctx.Arg("dry-run") == "true"
	result["current_attachment_ids"] = releaseAttachmentIDs(target.Edit)
	result["file"] = map[string]interface{}{
		"path":        filePath,
		"asset_name":  uploadName,
		"size_bytes":  size,
		"description": strings.TrimSpace(ctx.Arg("description")),
	}
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(result)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open asset file: %w", err)
	}
	defer file.Close()

	fields := map[string]string{}
	if description := strings.TrimSpace(ctx.Arg("description")); description != "" {
		fields["description"] = description
	}
	uploadEnv, err := ctx.PostMultipart("/attachments", fields, []client.MultipartFile{
		{
			FieldName: "file",
			FileName:  uploadName,
			Reader:    file,
		},
	})
	if err != nil {
		return fmt.Errorf("upload asset: %w", err)
	}

	attachment, err := releaseMap(uploadEnv.Data, "attachment upload response")
	if err != nil {
		return err
	}
	attachmentID := releaseIDString(attachment["id"])
	if attachmentID == "" {
		return fmt.Errorf("attachment upload response did not include an attachment ID")
	}

	nextIDs, _ := mergeReleaseAttachmentIDs(releaseAttachmentIDs(target.Edit), []string{attachmentID})
	releaseEnv, err := updateReleaseAttachments(ctx, target.VersionID, target.Edit, nextIDs)
	if err != nil {
		if cleanupErr := deleteAttachment(ctx, attachmentID); cleanupErr != nil {
			return fmt.Errorf("attach uploaded asset to release: %w (cleanup failed: %v)", err, cleanupErr)
		}
		return fmt.Errorf("attach uploaded asset to release: %w", err)
	}

	result["attachment_ids"] = nextIDs
	result["uploaded_attachment_id"] = attachmentID
	result["attachment"] = attachment
	result["release"] = releaseEnv.Data
	return ctx.OutputData(result)
}

func resolveReleaseTarget(ctx *common.RuntimeContext, identifier string) (*releaseTarget, error) {
	view, err := fetchReleaseView(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}

	versionID := releaseVersionID(identifier, view)
	if versionID == "" {
		return nil, fmt.Errorf("failed to resolve release version ID from %q", identifier)
	}

	edit, err := fetchReleaseEdit(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("fetch release edit data: %w", err)
	}

	return &releaseTarget{
		Identifier: identifier,
		VersionID:  versionID,
		View:       view,
		Edit:       edit,
	}, nil
}

func fetchReleaseView(ctx *common.RuntimeContext, id string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
	if err != nil {
		return nil, err
	}
	return releaseMap(env.Data, "release data")
}

func updateReleaseAttachments(ctx *common.RuntimeContext, versionID string, current map[string]interface{}, attachmentIDs []string) (*output.Envelope, error) {
	payload, err := releaseCurrentPayload(current, attachmentIDs)
	if err != nil {
		return nil, err
	}
	return ctx.CallAPI("PUT", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), versionID), payload)
}

func releaseActionResult(ctx *common.RuntimeContext, target *releaseTarget, action string) map[string]interface{} {
	return map[string]interface{}{
		"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"release_ref":  target.Identifier,
		"release_id":   target.VersionID,
		"tag_name":     releaseString(target.View, "tag_name"),
		"release_name": firstReleaseValue(releaseString(target.View, "name"), releaseString(target.Edit, "name")),
		"action":       action,
	}
}

func releaseVersionID(identifier string, view map[string]interface{}) string {
	if id := releaseIDString(view["version_id"]); id != "" {
		return id
	}
	if isNumericReleaseID(identifier) {
		return strings.TrimSpace(identifier)
	}
	return ""
}

func isNumericReleaseID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func releaseMap(data interface{}, name string) (map[string]interface{}, error) {
	result, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse %s", name)
	}
	return result, nil
}

func releaseCurrentPayload(current map[string]interface{}, attachmentIDs []string) (map[string]interface{}, error) {
	name := releaseString(current, "name")
	if name == "" {
		return nil, fmt.Errorf("required release name is missing in remote data")
	}
	tag := releaseString(current, "tag_name")
	if tag == "" {
		return nil, fmt.Errorf("required release tag is missing in remote data")
	}

	ids := make([]string, len(attachmentIDs))
	copy(ids, attachmentIDs)
	return map[string]interface{}{
		"name":             name,
		"tag_name":         tag,
		"body":             releaseString(current, "body"),
		"target_commitish": releaseString(current, "target_commitish"),
		"draft":            releaseBoolValue(current, "draft", false),
		"prerelease":       releaseBoolValue(current, "prerelease", false),
		"attachment_ids":   ids,
	}, nil
}

func releaseAttachments(values map[string]interface{}) []map[string]interface{} {
	if values == nil {
		return nil
	}

	switch attachments := values["attachments"].(type) {
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(attachments))
		for _, attachment := range attachments {
			item, ok := attachment.(map[string]interface{})
			if !ok {
				continue
			}
			items = append(items, item)
		}
		return items
	case []map[string]interface{}:
		return attachments
	default:
		return nil
	}
}

func mergeReleaseAttachmentIDs(currentIDs, requestedIDs []string) ([]string, []string) {
	merged := make([]string, 0, len(currentIDs)+len(requestedIDs))
	seen := map[string]bool{}

	for _, id := range currentIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		merged = append(merged, id)
	}

	added := make([]string, 0, len(requestedIDs))
	for _, id := range requestedIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		merged = append(merged, id)
		added = append(added, id)
	}

	return merged, added
}

func removeReleaseAttachmentIDs(currentIDs, requestedIDs []string) ([]string, []string) {
	removeSet := map[string]bool{}
	for _, id := range requestedIDs {
		removeSet[id] = true
	}

	remaining := make([]string, 0, len(currentIDs))
	removed := make([]string, 0, len(requestedIDs))
	for _, id := range currentIDs {
		if removeSet[id] {
			removed = append(removed, id)
			continue
		}
		remaining = append(remaining, id)
	}

	return remaining, removed
}

func resolveUploadFile(path, assetName string) (string, string, int64, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "." || cleanPath == "" {
		return "", "", 0, fmt.Errorf("--file is required")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("stat asset file: %w", err)
	}
	if info.IsDir() {
		return "", "", 0, fmt.Errorf("--file must point to a file, got directory %q", cleanPath)
	}

	uploadName := strings.TrimSpace(assetName)
	if uploadName == "" {
		uploadName = filepath.Base(cleanPath)
	}
	if strings.ContainsAny(uploadName, `/\`) {
		return "", "", 0, fmt.Errorf("--asset-name must be a filename, got %q", uploadName)
	}

	return cleanPath, uploadName, info.Size(), nil
}

func deleteAttachment(ctx *common.RuntimeContext, attachmentID string) error {
	_, err := ctx.CallAPI("DELETE", fmt.Sprintf("/attachments/%s", attachmentID), nil)
	return err
}
