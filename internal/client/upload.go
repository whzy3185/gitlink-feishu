package client

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
)

// progressThreshold is the minimum file size for which upload progress is
// reported on stderr.
const progressThreshold = 1 << 20 // 1 MiB

// progressReporter prints upload progress to stderr at 10% steps for files
// larger than progressThreshold. It implements io.Writer so it can sit on
// the tee side of the upload stream.
type progressReporter struct {
	name     string
	total    int64
	done     int64
	lastPct  int64
	lastLine int
	out      io.Writer
}

func newProgressReporter(name string, total int64) *progressReporter {
	return &progressReporter{name: name, total: total, lastPct: -1, out: os.Stderr}
}

func newProgressReporterTo(name string, total int64, out io.Writer) *progressReporter {
	return &progressReporter{name: name, total: total, lastPct: -1, out: out}
}

func (p *progressReporter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.total >= progressThreshold {
		pct := p.done * 100 / p.total
		if pct/10 > p.lastPct/10 || (pct == 100 && p.lastPct != 100) {
			line := fmt.Sprintf("uploading %s: %d%% (%s / %s)", p.name, pct, formatBytes(p.done), formatBytes(p.total))
			if pad := p.lastLine - len(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			fmt.Fprintf(p.out, "\r%s", line)
			p.lastLine = len(line)
			if pct >= 100 {
				fmt.Fprintln(p.out)
			}
			p.lastPct = pct
		}
	}
	return len(b), nil
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// PostMultipartFile uploads a local file as a multipart/form-data request.
// fileField is the form field name for the file (GitLink expects "file");
// extra fields (e.g. description) are added as plain form values.
func (c *Client) PostMultipartFile(path, filePath, fileField string, fields map[string]string) (*output.Envelope, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open upload file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat upload file: %w", err)
	}

	// Stream the multipart body through a pipe so arbitrarily large files
	// are never buffered in memory.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	progress := newProgressReporter(filepath.Base(filePath), info.Size())
	go func() {
		part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, io.TeeReader(f, progress)); err != nil {
			pw.CloseWithError(fmt.Errorf("read upload file: %w", err))
			return
		}
		for k, v := range fields {
			if v != "" {
				if err := writer.WriteField(k, v); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
		}
		pw.CloseWithError(writer.Close())
	}()

	fullURL := c.BaseURL + normalizeAPIPath(c.BaseURL, path)
	req, err := http.NewRequest("POST", fullURL, pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if c.Debug {
		fmt.Printf("→ POST %s (multipart, %s)\n", fullURL, filepath.Base(filePath))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if c.Debug {
		fmt.Printf("← %d %s\n", resp.StatusCode, string(respData[:min(len(respData), 200)]))
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData))),
		}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respData, &raw); err != nil {
		return output.SuccessEnvelope(string(respData), nil), nil
	}
	if status, ok := raw["status"].(float64); ok && status != 0 && status != 200 && status != 201 && status != 1 {
		msg, _ := raw["message"].(string)
		return output.ErrorEnvelope(int(status), msg, ""), &APIError{
			StatusCode: int(status),
			Code:       int(status),
			Message:    msg,
		}
	}
	return output.SuccessEnvelope(raw, nil), nil
}

// DownloadFile streams a GET response body to destPath and returns the
// number of bytes written.
func (c *Client) DownloadFile(path, destPath string) (int64, error) {
	fullURL := c.BaseURL + normalizeAPIPath(c.BaseURL, path)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return 0, err
	}
	if c.Debug {
		fmt.Printf("→ GET %s (download to %s)\n", fullURL, destPath)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return 0, &APIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data))),
		}
	}
	// Unknown attachment ids fall through to the web frontend, which answers
	// 200 with an HTML page; surface that as an error instead of saving it.
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		return 0, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "non_api_response",
			Message:    "endpoint returned an HTML page instead of file data; check the attachment id",
		}
	}
	// Deleted/unknown attachments answer 200 with a JSON error body
	// ({"status":404,"message":"..."}); surface that as an error too.
	if strings.Contains(ct, "application/json") {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var body struct {
			Status  float64 `json:"status"`
			Message string  `json:"message"`
		}
		if err := json.Unmarshal(data, &body); err == nil && body.Status != 0 && body.Status != 200 && body.Status != 201 && body.Status != 1 {
			return 0, &APIError{
				StatusCode: int(body.Status),
				Code:       int(body.Status),
				Message:    body.Message,
			}
		}
		return 0, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "non_file_response",
			Message:    "endpoint returned JSON instead of file data: " + strings.TrimSpace(string(data)),
		}
	}

	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return n, fmt.Errorf("write output file: %w", err)
	}
	return n, nil
}
