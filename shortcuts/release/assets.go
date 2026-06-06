package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type releaseAsset struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Filesize    string `json:"filesize,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

func runReleaseAssets(ctx *common.RuntimeContext) error {
	release, err := fetchRelease(ctx)
	if err != nil {
		return err
	}
	return ctx.Output(output.SuccessEnvelope(releaseAssetsPayload(release), nil))
}

func runReleaseDownload(ctx *common.RuntimeContext) error {
	release, err := fetchRelease(ctx)
	if err != nil {
		return err
	}
	sourceURL, filename, sourceType, err := selectDownloadSource(ctx, release)
	if err != nil {
		return err
	}

	result, err := ctx.Download(sourceURL)
	if err != nil {
		return err
	}
	outPath, err := resolveOutputPath(ctx.Arg("output"), filename)
	if err != nil {
		return err
	}
	if ctx.Arg("force") != "true" {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("output file already exists: %s (use --force to overwrite)", outPath)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, result.Data, 0644); err != nil {
		return err
	}

	return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
		"path":         outPath,
		"bytes":        len(result.Data),
		"source":       sourceURL,
		"source_type":  sourceType,
		"content_type": result.ContentType,
	}, nil))
}

func fetchRelease(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return nil, err
	}
	id, _ := ctx.RequireArg("id")
	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected release response shape")
	}
	return data, nil
}

func releaseAssetsPayload(release map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"version_id":  stringValue(release["version_id"]),
		"tag_name":    stringValue(release["tag_name"]),
		"name":        stringValue(release["name"]),
		"attachments": releaseAttachments(release),
		"archives": map[string]string{
			"zip": stringValue(release["zipball_url"]),
			"tar": stringValue(release["tarball_url"]),
		},
	}
}

func selectDownloadSource(ctx *common.RuntimeContext, release map[string]interface{}) (string, string, string, error) {
	archive := strings.ToLower(strings.TrimSpace(ctx.Arg("archive")))
	assetSelector := strings.TrimSpace(ctx.Arg("asset"))
	if archive != "" && assetSelector != "" {
		return "", "", "", fmt.Errorf("--asset and --archive cannot be used together")
	}
	if archive != "" {
		return selectArchiveSource(ctx, release, archive)
	}

	assets := releaseAttachments(release)
	asset, err := selectAsset(assets, assetSelector)
	if err != nil {
		return "", "", "", err
	}
	return asset.URL, safeFilename(asset.Title, "attachment-"+asset.ID), "asset", nil
}

func selectArchiveSource(ctx *common.RuntimeContext, release map[string]interface{}, archive string) (string, string, string, error) {
	tag := stringValue(release["tag_name"])
	if tag == "" {
		tag = ctx.Arg("id")
	}
	switch archive {
	case "zip":
		u := stringValue(release["zipball_url"])
		if u == "" {
			return "", "", "", fmt.Errorf("release does not provide zipball_url")
		}
		return u, safeFilename(fmt.Sprintf("%s-%s.zip", ctx.Repo, tag), "release.zip"), "archive", nil
	case "tar", "tar.gz", "tgz":
		u := stringValue(release["tarball_url"])
		if u == "" {
			return "", "", "", fmt.Errorf("release does not provide tarball_url")
		}
		return u, safeFilename(fmt.Sprintf("%s-%s.tar.gz", ctx.Repo, tag), "release.tar.gz"), "archive", nil
	default:
		return "", "", "", fmt.Errorf("--archive must be zip or tar")
	}
}

func selectAsset(assets []releaseAsset, selector string) (releaseAsset, error) {
	if len(assets) == 0 {
		return releaseAsset{}, fmt.Errorf("release has no attachments")
	}
	if selector == "" {
		if len(assets) == 1 {
			return assets[0], nil
		}
		return releaseAsset{}, fmt.Errorf("release has multiple attachments; specify --asset by id or title")
	}
	for _, asset := range assets {
		if asset.ID == selector || asset.Title == selector {
			return asset, nil
		}
	}
	return releaseAsset{}, fmt.Errorf("release attachment %q not found", selector)
}

func releaseAttachments(release map[string]interface{}) []releaseAsset {
	raw, ok := release["attachments"].([]interface{})
	if !ok {
		return nil
	}
	assets := make([]releaseAsset, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		assets = append(assets, releaseAsset{
			ID:          stringValue(m["id"]),
			Title:       stringValue(m["title"]),
			Filesize:    stringValue(m["filesize"]),
			Description: stringValue(m["description"]),
			URL:         stringValue(m["url"]),
		})
	}
	return assets
}

func resolveOutputPath(out, filename string) (string, error) {
	if strings.TrimSpace(out) == "" {
		out = "."
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, filename), nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if strings.HasSuffix(out, string(os.PathSeparator)) || strings.HasSuffix(out, "/") {
		return filepath.Join(out, filename), nil
	}
	return out, nil
}

func safeFilename(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "." || name == string(os.PathSeparator) || name == "" {
		return fallback
	}
	return name
}

func stringValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return ""
	}
}
