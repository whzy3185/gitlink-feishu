package workflow

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAPIInt(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want int
	}{
		{"int", 42, 42},
		{"int64", int64(100), 100},
		{"float64", 3.14, 3},
		{"float64 int", 99.0, 99},
		{"string int", "55", 55},
		{"string empty", "", 0},
		{"string float", "3.14", 3},
		{"bool", true, 0},
		{"nil", nil, 0},
		{"json number", json.Number("123"), 123},
		{"uint", uint(10), 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiInt(tt.v); got != tt.want {
				t.Fatalf("apiInt = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAPIString(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"string", "hello", "hello"},
		// trimTrailingZero only strips exactly 6 trailing zeros;
		// 3.14 has only 4 zeros after 2 significant digits
		{"float64", float64(3.14), "3.140000"},
		// 42.0 has all 6 trailing zeros, so they get stripped
		{"float64 int", float64(42.0), "42"},
		{"int", 42, "42"},
		{"int64", int64(100), "100"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, ""},
		{"json number", json.Number("99"), "99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiString(tt.v)
			if got != tt.want {
				t.Fatalf("apiString = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIBool(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"string true", "true", true},
		{"string false", "false", false},
		// strconv.ParseBool is case-insensitive
		{"string TRUE", "TRUE", true},
		// TrimSpace is applied internally
		{"string with spaces", "  true  ", true},
		{"string yes", "yes", false},
		{"int 1", 1, true},
		{"int 0", 0, false},
		{"float64 1", 1.0, true},
		{"float64 0", 0.0, false},
		{"nil", nil, false},
		{"json number 1", json.Number("1"), true},
		{"json number 0", json.Number("0"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiBool(tt.v); got != tt.want {
				t.Fatalf("apiBool = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPITime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		v    interface{}
		zero bool
	}{
		{"time.Time", now, false},
		{"rfc3339", "2024-01-15T10:30:00Z", false},
		{"rfc3339 nano", "2024-01-15T10:30:00.123456789Z", false},
		{"date only", "2024-01-15", false},
		{"unix seconds", int64(1705312200), false},
		// 1705312200000 > 1e12, treated as milliseconds-since-epoch → valid time
		{"unix millis", int64(1705312200000), false},
		{"float64 seconds", 1705312200.0, false},
		{"empty string", "", true},
		{"nil", nil, true},
		{"invalid", "not-a-time", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiTime(tt.v)
			if tt.zero && !got.IsZero() {
				t.Fatalf("expected zero time, got %v", got)
			}
			if !tt.zero && got.IsZero() {
				t.Fatal("expected non-zero time")
			}
		})
	}
}

func TestAPITimeRFC3339(t *testing.T) {
	got := apiTime("2024-06-01T12:00:00Z")
	if got.Year() != 2024 || got.Month() != 6 || got.Day() != 1 {
		t.Fatalf("apiTime parsed incorrectly: %v", got)
	}
}

func TestAPIStringSlice(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want []string
	}{
		{"string slice", []string{"a", "b"}, []string{"a", "b"}},
		{"interface slice", []interface{}{"a", "b"}, []string{"a", "b"}},
		{"comma string", "a, b, c", []string{"a", "b", "c"}},
		{"empty string", "", nil},
		{"nil", nil, nil},
		{"single string", "only", []string{"only"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiStringSlice(tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("apiStringSlice len = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("apiStringSlice[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAPIStringValue(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"string", "hello", "hello"},
		{"map with name", map[string]interface{}{"name": "testname"}, "testname"},
		{"map with title", map[string]interface{}{"title": "testtitle"}, "testtitle"},
		{"map with login", map[string]interface{}{"login": "testlogin"}, "testlogin"},
		{"map with label", map[string]interface{}{"label": "testlabel"}, "testlabel"},
		{"empty map", map[string]interface{}{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiStringValue(tt.v); got != tt.want {
				t.Fatalf("apiStringValue = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIAuthor(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"login", map[string]interface{}{"login": "user1"}, "user1"},
		{"name", map[string]interface{}{"name": "User Name"}, "User Name"},
		{"username", map[string]interface{}{"username": "uname"}, "uname"},
		{"full_name", map[string]interface{}{"full_name": "Full Name"}, "Full Name"},
		{"display_name", map[string]interface{}{"display_name": "Display"}, "Display"},
		{"string", "directstring", "directstring"},
		{"empty map", map[string]interface{}{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiAuthor(tt.v); got != tt.want {
				t.Fatalf("apiAuthor = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPILatestTime(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	got := apiLatestTime(t1, t2, t3)
	if !got.Equal(t2) {
		t.Fatalf("apiLatestTime = %v, want %v", got, t2)
	}

	got = apiLatestTime()
	if !got.IsZero() {
		t.Fatal("expected zero time for no args")
	}

	got = apiLatestTime(time.Time{}, t1, time.Time{})
	if !got.Equal(t1) {
		t.Fatalf("apiLatestTime with zeros = %v, want %v", got, t1)
	}
}

func TestAPIAgeInDays(t *testing.T) {
	if got := apiAgeInDays(time.Time{}); got != -1 {
		t.Fatalf("apiAgeInDays zero = %d, want -1", got)
	}

	recent := time.Now().Add(-24 * time.Hour)
	if got := apiAgeInDays(recent); got != 1 {
		t.Fatalf("apiAgeInDays 24h ago = %d, want 1", got)
	}
}

func TestAPIObject(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		nil  bool
	}{
		{"map", map[string]interface{}{"key": "val"}, false},
		{"nil", nil, true},
		{"string json object", `{"key":"val"}`, false},
		{"empty string", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiObject(tt.data)
			if tt.nil && got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
			if !tt.nil && got == nil {
				t.Fatal("expected non-nil")
			}
		})
	}
}

func TestAPIList(t *testing.T) {
	tests := []struct {
		name   string
		data   interface{}
		length int
	}{
		{"slice", []interface{}{map[string]interface{}{"id": float64(1)}}, 1},
		{"nil", nil, 0},
		{"json string array", `[{"id":1}]`, 1},
		{"empty string", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiList(tt.data)
			if len(got) != tt.length {
				t.Fatalf("apiList len = %d, want %d", len(got), tt.length)
			}
		})
	}
}

func TestAPINormalizeData(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		isNil   bool
		isMap   bool
		isSlice bool
	}{
		{"nil", nil, true, false, false},
		{"empty string", "", true, false, false},
		{"string json object", `{"a":"b"}`, false, true, false},
		{"string json array", `[1,2]`, false, false, true},
		{"plain string", "hello", false, false, false},
		{"map", map[string]interface{}{"a": "b"}, false, true, false},
		{"slice", []interface{}{1, 2}, false, false, true},
		{"json raw message object", json.RawMessage(`{"a":"b"}`), false, true, false},
		{"json raw message array", json.RawMessage(`[1,2]`), false, false, true},
		{"empty json raw", json.RawMessage{}, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAPIData(tt.data)
			if err != nil {
				t.Fatalf("normalizeAPIData error: %v", err)
			}
			if tt.isNil && got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
			if tt.isMap {
				if _, ok := got.(map[string]interface{}); !ok {
					t.Fatalf("expected map, got %T", got)
				}
			}
			if tt.isSlice {
				if _, ok := got.([]interface{}); !ok {
					t.Fatalf("expected slice, got %T", got)
				}
			}
			if !tt.isNil && !tt.isMap && !tt.isSlice {
				if _, ok := got.(string); !ok {
					t.Fatalf("expected string, got %T", got)
				}
			}
		})
	}
}

func TestWorkflowRepoPath(t *testing.T) {
	got := workflowRepoPath("owner", "repo")
	if got != "/v1/owner/repo" {
		t.Fatalf("workflowRepoPath = %q, want /v1/owner/repo", got)
	}
}

func TestLooksLikeIssueOrRepoItem(t *testing.T) {
	tests := []struct {
		name string
		v    map[string]interface{}
		want bool
	}{
		{"with title", map[string]interface{}{"title": "test"}, true},
		{"with subject", map[string]interface{}{"subject": "test"}, true},
		{"with number", map[string]interface{}{"number": float64(1)}, true},
		{"with id", map[string]interface{}{"id": float64(1)}, true},
		{"with iid", map[string]interface{}{"iid": float64(1)}, true},
		{"with issue_number", map[string]interface{}{"issue_number": float64(1)}, true},
		{"with project_issues_index", map[string]interface{}{"project_issues_index": float64(1)}, true},
		{"empty", map[string]interface{}{}, false},
		{"other keys", map[string]interface{}{"foo": "bar"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeIssueOrRepoItem(tt.v); got != tt.want {
				t.Fatalf("looksLikeIssueOrRepoItem = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimTrailingZero(t *testing.T) {
	// trimTrailingZero only strips exactly 6 trailing "0" characters,
	// then ".000000", then ".0", then "."
	tests := []struct {
		input string
		want  string
	}{
		{"42.000000", "42"},
		// 42.500000 has only 4 trailing zeros (after "5"), so nothing stripped
		{"42.500000", "42.500000"},
		{"42.0", "42"},
		{"42.", "42"},
		{"42", "42"},
		// 42.100000 has only 4 trailing zeros (after "1"), so nothing stripped
		{"42.100000", "42.100000"},
		{".0", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := trimTrailingZero(tt.input); got != tt.want {
				t.Fatalf("trimTrailingZero = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountStaleItems(t *testing.T) {
	items := []map[string]interface{}{
		{"updated_at": time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)},
		{"updated_at": time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339)},
		{"updated_at": time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339)},
	}
	got := countStaleItems(items, 30)
	if got != 1 {
		t.Fatalf("countStaleItems = %d, want 1", got)
	}

	got = countStaleItems(items, 7)
	if got != 2 {
		t.Fatalf("countStaleItems(7) = %d, want 2", got)
	}
}

func TestItemActivityTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	older := now.Add(-10 * 24 * time.Hour)
	item := map[string]interface{}{
		"updated_at": older.Format(time.RFC3339),
		"created_at": now.Format(time.RFC3339),
	}
	got := itemActivityTime(item)
	if !got.Equal(now) {
		t.Fatalf("itemActivityTime should return latest = %v, got %v", now, got)
	}

	if got := itemActivityTime(nil); !got.IsZero() {
		t.Fatal("expected zero time for nil item")
	}
}

func TestCloneValues(t *testing.T) {
	orig := map[string][]string{"key": {"val1", "val2"}}
	cloned := cloneValues(orig)
	cloned["key"][0] = "modified"
	if orig["key"][0] != "val1" {
		t.Fatal("cloneValues did not deep copy")
	}

	if got := cloneValues(nil); got == nil {
		t.Fatal("expected non-nil for nil input")
	}
	if len(cloneValues(nil)) != 0 {
		t.Fatal("expected empty values for nil input")
	}
}

func TestBuildPassing(t *testing.T) {
	tests := []struct {
		name string
		item map[string]interface{}
		want bool
	}{
		{"status success", map[string]interface{}{"status": "success"}, true},
		{"status passed", map[string]interface{}{"status": "passed"}, true},
		{"status failed", map[string]interface{}{"status": "failed"}, false},
		{"state success", map[string]interface{}{"state": "success"}, true},
		{"result passed", map[string]interface{}{"result": "passed"}, true},
		{"conclusion ok", map[string]interface{}{"conclusion": "ok"}, true},
		{"status_text done", map[string]interface{}{"status_text": "done"}, true},
		{"build passed", map[string]interface{}{"status": "build passed"}, true},
		{"canceled", map[string]interface{}{"status": "canceled"}, false},
		{"running", map[string]interface{}{"status": "running"}, false},
		{"pending", map[string]interface{}{"status": "pending"}, false},
		{"success bool", map[string]interface{}{"success": true}, true},
		{"empty", map[string]interface{}{}, false},
		{"unknown", map[string]interface{}{"status": "unknown_status"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildPassing(tt.item); got != tt.want {
				t.Fatalf("buildPassing = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueScoringNotes(t *testing.T) {
	notes := []ScoringNote{
		{Metric: "ci", Note: "failed"},
		{Metric: "ci", Note: "failed"},
		{Metric: "release", Note: "missing"},
	}
	got := uniqueScoringNotes(notes)
	if len(got) != 2 {
		t.Fatalf("uniqueScoringNotes len = %d, want 2", len(got))
	}
}

func TestQueryWithPageLimit(t *testing.T) {
	q := queryWithPageLimit(nil, 2, 50)
	if q.Get("page") != "2" || q.Get("limit") != "50" {
		t.Fatalf("queryWithPageLimit = %v", q)
	}

	q = queryWithPageLimit(nil, 0, 0)
	if q.Get("page") != "" && q.Get("limit") != "" {
		t.Fatalf("expected empty page/limit for zero values")
	}
}

func TestIssueListQuery(t *testing.T) {
	q := issueListQuery("open")
	if q.Get("state") != "open" {
		t.Fatalf("issueListQuery state = %q", q.Get("state"))
	}
}

func TestParseAPITimeString(t *testing.T) {
	tests := []struct {
		name  string
		value string
		zero  bool
	}{
		{"rfc3339", "2024-01-15T10:30:00Z", false},
		{"rfc3339 nano", "2024-01-15T10:30:00.123456789Z", false},
		{"no T", "2024-01-15 10:30:00", false},
		{"date only", "2024-01-15", false},
		{"unix seconds", "1705312200", false},
		{"empty", "", true},
		{"garbage", "not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAPIStringTime(tt.value)
			if tt.zero && !got.IsZero() {
				t.Fatalf("expected zero, got %v", got)
			}
			if !tt.zero && got.IsZero() {
				t.Fatal("expected non-zero")
			}
		})
	}
}

func TestParseAPINumericTime(t *testing.T) {
	if got := parseAPINumericTime(0); !got.IsZero() {
		t.Fatal("expected zero for 0")
	}
	if got := parseAPINumericTime(-1); !got.IsZero() {
		t.Fatal("expected zero for negative")
	}
	// Unix seconds (< 1e12)
	if got := parseAPINumericTime(1705312200); got.IsZero() {
		t.Fatal("expected non-zero for unix seconds")
	}
	// > 1e12 is treated as milliseconds; this results in a valid time
	if got := parseAPINumericTime(1705312200000); got.IsZero() {
		t.Fatal("expected non-zero for millisecond timestamp")
	}
}

func TestLatestTimeFromItems(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	items := []map[string]interface{}{
		{"updated_at": t1.Format(time.RFC3339)},
		{"updated_at": t2.Format(time.RFC3339)},
	}
	got := latestTimeFromItems(items)
	if !got.Equal(t2) {
		t.Fatalf("latestTimeFromItems = %v, want %v", got, t2)
	}

	if got := latestTimeFromItems(nil); !got.IsZero() {
		t.Fatal("expected zero for nil")
	}
}

func TestUpdateRecentActivity(t *testing.T) {
	t1 := time.Now().Add(-5 * 24 * time.Hour).Truncate(time.Second)
	t2 := time.Now().Add(-2 * 24 * time.Hour).Truncate(time.Second)

	input := HealthInput{}
	known, _, input := updateRecentActivity(input, t1)
	if !known {
		t.Fatalf("first update: known=%v, want true", known)
	}

	// Update with more recent
	known, days, input := updateRecentActivity(input, t2)
	if !known || days > 3 {
		t.Fatalf("second update: known=%v days=%d, want true and days <= 3", known, days)
	}

	// Zero time should not change
	known2, days2, _ := updateRecentActivity(input, time.Time{})
	if !known2 || days2 != days {
		t.Fatalf("zero time update should not change: known=%v days=%d", known2, days2)
	}
}

func TestAPIIntStringFallback(t *testing.T) {
	if got := apiInt("notanumber"); got != 0 {
		t.Fatalf("apiInt invalid string = %d, want 0", got)
	}
}

func TestAPIAuthorPriority(t *testing.T) {
	got := apiAuthor(map[string]interface{}{
		"name":  "Second",
		"login": "First",
	})
	if got != "First" {
		t.Fatalf("apiAuthor priority = %q, want First", got)
	}
}

func TestAPIStringSliceMapItems(t *testing.T) {
	got := apiStringSlice([]interface{}{
		map[string]interface{}{"name": "item1"},
		map[string]interface{}{"title": "item2"},
	})
	if len(got) != 2 || got[0] != "item1" || got[1] != "item2" {
		t.Fatalf("apiStringSlice map items = %v", got)
	}
}

func TestAPIObjectSingleItemArray(t *testing.T) {
	got := apiObject([]interface{}{
		map[string]interface{}{"key": "val"},
	})
	if got == nil {
		t.Fatal("expected single item from array")
	}
	if got["key"] != "val" {
		t.Fatalf("unexpected value: %v", got)
	}
}

func TestAPIListNestedKeys(t *testing.T) {
	got := apiList(map[string]interface{}{
		"issues": []interface{}{
			map[string]interface{}{"id": float64(1)},
		},
	})
	if len(got) != 1 {
		t.Fatalf("apiList nested issues len = %d, want 1", len(got))
	}
}

func TestAPIListLooksLikeItem(t *testing.T) {
	got := apiList(map[string]interface{}{
		"title": "test",
		"id":    float64(1),
	})
	if len(got) != 1 {
		t.Fatalf("apiList looksLikeItem len = %d, want 1", len(got))
	}
}

func TestAPIStringJSONNumber(t *testing.T) {
	if got := apiString(json.Number("42")); got != "42" {
		t.Fatalf("apiString json.Number = %q, want 42", got)
	}
}

func TestAPITimeFromJSONNumber(t *testing.T) {
	got := apiTime(json.Number("1705312200"))
	if got.IsZero() {
		t.Fatal("expected non-zero from json.Number")
	}

	got = apiTime(json.Number("notanumber"))
	if !got.IsZero() {
		t.Fatal("expected zero for invalid json.Number")
	}
}

func TestAPIAgeInDaysBoundary(t *testing.T) {
	recent := time.Now().Add(-23 * time.Hour)
	if got := apiAgeInDays(recent); got != 0 {
		t.Fatalf("apiAgeInDays 23h = %d, want 0", got)
	}
}

func TestCountStaleItemsDefaultDays(t *testing.T) {
	items := []map[string]interface{}{
		{"updated_at": time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)},
	}
	if got := countStaleItems(items, 0); got != 1 {
		t.Fatalf("countStaleItems default days = %d, want 1", got)
	}
	if got := countStaleItems(items, -5); got != 1 {
		t.Fatalf("countStaleItems negative days = %d, want 1", got)
	}
}

func TestParseAPIStringTimeWithT(t *testing.T) {
	got := parseAPIStringTime("2024-06-15T08:30:00")
	if got.IsZero() {
		t.Fatal("expected non-zero for time with T separator")
	}
}

func TestNormalizeAPIDataBytes(t *testing.T) {
	got, err := normalizeAPIData(json.RawMessage(`"hello"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := got.(string)
	if !ok || s != "hello" {
		t.Fatalf("expected 'hello', got %v", got)
	}
}

func TestNormalizeAPIDataInvalidJSON(t *testing.T) {
	got, err := normalizeAPIData("{invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{invalid" {
		t.Fatalf("expected raw string, got %v", got)
	}
}

func TestBuildPassingCaseInsensitive(t *testing.T) {
	got := buildPassing(map[string]interface{}{"status": "SUCCESS"})
	if !got {
		t.Fatal("expected true for uppercase SUCCESS")
	}

	got = buildPassing(map[string]interface{}{"status": "  success  "})
	if !got {
		t.Fatal("expected true for whitespace-padded status")
	}
}

func TestAPIStringSliceWithStrings(t *testing.T) {
	got := apiStringSlice([]string{"a", "b", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
}

func TestAPIStringSliceEmptyParts(t *testing.T) {
	got := apiStringSlice("a,,b")
	if len(got) != 2 {
		t.Fatalf("expected 2 parts, got %d: %v", len(got), got)
	}
}

func TestAPIStringAllTypes(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"float32", float32(42), "42"},
		{"int32", int32(42), "42"},
		{"uint64", uint64(100), "100"},
		{"uint32", uint32(50), "50"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiString(tt.v); got != tt.want {
				t.Fatalf("apiString(%v) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestAPIIntAllTypes(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want int
	}{
		{"int8", int8(8), 8},
		{"int16", int16(16), 16},
		{"int32", int32(32), 32},
		{"uint8", uint8(8), 8},
		{"uint16", uint16(16), 16},
		{"uint32", uint32(32), 32},
		{"uint64", uint64(64), 64},
		{"float32", float32(42.0), 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiInt(tt.v); got != tt.want {
				t.Fatalf("apiInt(%v) = %d, want %d", tt.v, got, tt.want)
			}
		})
	}
}

func TestWorkflowRepoPathTrims(t *testing.T) {
	got := workflowRepoPath("  owner  ", "  repo  ")
	if !strings.Contains(got, "/v1/owner/repo") {
		t.Fatalf("expected trimmed path, got %q", got)
	}
}
