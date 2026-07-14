// Package attachment implements shortcuts for uploading and downloading
// platform attachments (release assets, issue attachments, etc.).
//
// Upload wraps the multipart POST /api/attachments endpoint and returns the
// attachment id that can be fed to `release +create --attachment-ids`;
// download wraps GET /api/attachments/:uuid and streams the file to
// disk, giving the CLI a scriptable path for large-file transfer instead of
// the web UI.
package attachment

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// splitFiles parses a comma-separated file list, trimming blanks.
func splitFiles(arg string) []string {
	var files []string
	for _, f := range strings.Split(arg, ",") {
		if f = strings.TrimSpace(f); f != "" {
			files = append(files, f)
		}
	}
	return files
}

// uploadConcurrently uploads several files with a bounded worker pool.
// Per-byte progress is suppressed (interleaved lines would garble); instead
// one line per completed file goes to stderr. Results keep input order.
func uploadConcurrently(ctx *common.RuntimeContext, files []string, fields map[string]string, concurrency int) ([]interface{}, error) {
	quiet := *ctx.Client
	quiet.NoProgress = true

	type result struct {
		env *output.Envelope
		err error
	}
	results := make([]result, len(files))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, file := range files {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			env, err := quiet.PostMultipartFile("/attachments", file, "file", fields)
			results[i] = result{env: env, err: err}
			mu.Lock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "uploaded %s: error: %v\n", filepath.Base(file), err)
			} else {
				fmt.Fprintf(os.Stderr, "uploaded %s\n", filepath.Base(file))
			}
			mu.Unlock()
		}(i, file)
	}
	wg.Wait()

	out := make([]interface{}, 0, len(files))
	for i, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("upload %q failed: %w", files[i], r.err)
		}
		out = append(out, map[string]interface{}{"file": files[i], "result": r.env.Data})
	}
	return out, nil
}

// Shortcuts returns attachment upload/download shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}

	return []*common.Shortcut{
		{
			Name:        "upload",
			Description: tr.T("cmd.attachment.upload.short"),
			Long:        tr.T("cmd.attachment.upload.long"),
			Flags: []common.Flag{
				{Name: "file", Short: "f", Usage: tr.T("flag.attachment.file"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.attachment.description")},
				{Name: "concurrency", Short: "c", Usage: tr.T("flag.attachment.concurrency"), Default: "3"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				fileArg, err := ctx.RequireArg("file")
				if err != nil {
					return err
				}
				files := splitFiles(fileArg)
				for _, file := range files {
					info, err := os.Stat(file)
					if err != nil {
						return fmt.Errorf("cannot access file %q: %w", file, err)
					}
					if info.IsDir() {
						return fmt.Errorf("%q is a directory, expected a file", file)
					}
				}
				fields := map[string]string{
					"description": ctx.Arg("description"),
				}
				if len(files) == 1 {
					env, err := ctx.Client.PostMultipartFile("/attachments", files[0], "file", fields)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				}
				concurrency, err := strconv.Atoi(ctx.Arg("concurrency"))
				if err != nil || concurrency < 1 {
					concurrency = 3
				}
				results, err := uploadConcurrently(ctx, files, fields, concurrency)
				if err != nil {
					return err
				}
				return ctx.OutputData(results)
			},
		},
		{
			Name:        "download",
			Description: tr.T("cmd.attachment.download.short"),
			Long:        tr.T("cmd.attachment.download.long"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.attachment.id"), Required: true},
				{Name: "output", Short: "o", Usage: tr.T("flag.attachment.output")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				dest := ctx.Arg("output")
				if dest == "" {
					dest = id
				}
				n, err := ctx.Client.DownloadFile("/attachments/"+id, dest)
				if err != nil {
					return err
				}
				return ctx.OutputData(map[string]interface{}{
					"file":  filepath.Clean(dest),
					"bytes": n,
				})
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.attachment.delete.short"),
			Long:        tr.T("cmd.attachment.delete.long"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.attachment.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.Client.Delete("/attachments/"+id, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
