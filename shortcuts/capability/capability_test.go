package capability

import (
	"testing"

	intcap "github.com/gitlink-org/gitlink-cli/internal/capability"
)

func TestBuildResultRowsOrderAndContent(t *testing.T) {
	results := map[string]*intcap.DomainStatus{
		"label":    {Status: intcap.StatusAvailable},
		"webhook":  {Status: intcap.StatusUnavailable, Message: "返回 HTML"},
		"pipeline": {Status: intcap.StatusError, Message: "网络错误"},
	}
	rows := buildResultRows(results)
	// buildResultRows always emits the full fixed domain catalog (11 entries).
	if len(rows) != 11 {
		t.Fatalf("got %d rows, want 11", len(rows))
	}
	wantFirst := "label"
	if rows[0].Domain != wantFirst {
		t.Errorf("first row = %q, want %q", rows[0].Domain, wantFirst)
	}
	byDomain := map[string]resultRow{}
	for _, r := range rows {
		byDomain[r.Domain] = r
	}
	if byDomain["label"].Status != "available" {
		t.Errorf("label status = %q", byDomain["label"].Status)
	}
	if byDomain["webhook"].Status != "unavailable" || byDomain["webhook"].Message != "返回 HTML" {
		t.Errorf("webhook row wrong: %+v", byDomain["webhook"])
	}
}

func TestBuildResultRowsMissingDomainIsSkipped(t *testing.T) {
	// An empty results map → every domain lands in the "skipped" branch.
	rows := buildResultRows(map[string]*intcap.DomainStatus{})
	for _, r := range rows {
		if r.Status != "unknown" {
			t.Errorf("domain %s: expected unknown/skipped, got %s", r.Domain, r.Status)
		}
	}
}

func TestBuildResultRowsNilEntryIsSkipped(t *testing.T) {
	results := map[string]*intcap.DomainStatus{"label": nil}
	rows := buildResultRows(results)
	for _, r := range rows {
		if r.Domain == "label" && r.Status != "unknown" {
			t.Errorf("nil entry should be skipped, got %s", r.Status)
		}
	}
}

func TestStatusHelpers(t *testing.T) {
	cases := []struct {
		status         intcap.Status
		str, text, ico string
	}{
		{intcap.StatusAvailable, "available", "可用 ✓", "✓"},
		{intcap.StatusUnavailable, "unavailable", "不可用 ✗", "✗"},
		{intcap.StatusError, "error", "错误 ✗", "✗"},
		{intcap.StatusUnknown, "unknown", "未知 ?", "?"},
	}
	for _, c := range cases {
		if got := statusString(c.status); got != c.str {
			t.Errorf("statusString(%v) = %q, want %q", c.status, got, c.str)
		}
		if got := statusText(c.status); got != c.text {
			t.Errorf("statusText(%v) = %q, want %q", c.status, got, c.text)
		}
		if got := statusEmoji(c.str); got != c.ico {
			t.Errorf("statusEmoji(%q) = %q, want %q", c.str, got, c.ico)
		}
	}
}
