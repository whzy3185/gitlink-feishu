package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeMember(t *testing.T, root, login string, infl, contrib, lang int, cats string, mirror int) string {
	dir := filepath.Join(root, login)
	writeFixture(t, dir, "develop.json", `{"ok":true,"data":{"platform":{"influence":99,"contribution":99,"activity":99,"experience":99,"language":98},"user":{"influence":`+itoa(infl)+`,"contribution":`+itoa(contrib)+`,"activity":80,"experience":85,"language":`+itoa(lang)+`,"languages_percent":{"Go":0.5,"Python":0.3},"each_language_score":{"Go":88,"Python":80}}}}`)
	writeFixture(t, dir, "major.json", `{"ok":true,"data":{"categories":[`+cats+`]}}`)
	writeFixture(t, dir, "role.json", `{"ok":true,"data":{"role":{"owner":{"count":5,"percent":0.8},"developer":{"count":2,"percent":0.2},"manager":{"count":0,"percent":0},"reporter":{"count":0,"percent":0}},"total_projects_count":7}}`)
	writeFixture(t, dir, "activity.json", `{"ok":true,"data":{"dates":["2026.06.13","2026.06.14"],"commits_count":[1,2],"issues_count":[0,1],"pull_requests_count":[0,0]}}`)
	writeFixture(t, dir, "contribution.json", `{"ok":true,"data":{"total_contributions":123,"headmaps":[{"date":"2026-06-14","contributions":5}]}}`)
	writeFixture(t, dir, "user.json", `{"ok":true,"data":{"login":"`+login+`","name":"`+login+`","real_name":"研究员`+login+`","custom_department":"某大学","mirror_projects_count":`+itoa(mirror)+`}}`)
	return dir
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func TestLoadSubject(t *testing.T) {
	root := t.TempDir()
	dir := makeMember(t, root, "alice", 70, 90, 66, `"人工智能","大数据"`, 0)
	s, err := loadSubject(dir)
	if err != nil {
		t.Fatalf("loadSubject: %v", err)
	}
	if s.Login != "alice" {
		t.Fatalf("login=%q", s.Login)
	}
	if s.Dev.User.Contribution != 90 {
		t.Fatalf("contribution=%d want 90", s.Dev.User.Contribution)
	}
	if s.Dev.Platform.Influence != 99 {
		t.Fatalf("platform influence=%d want 99", s.Dev.Platform.Influence)
	}
	if len(s.Major) != 2 || s.Major[0] != "人工智能" {
		t.Fatalf("major=%v", s.Major)
	}
	if s.Role.TotalProjectsCount != 7 {
		t.Fatalf("total projects=%d", s.Role.TotalProjectsCount)
	}
	if s.displayName() != "研究员alice" {
		t.Fatalf("displayName=%q", s.displayName())
	}
}

func TestAbilityGetAndAverages(t *testing.T) {
	root := t.TempDir()
	a, _ := loadSubject(makeMember(t, root, "a", 60, 80, 60, `"AI"`, 0))
	b, _ := loadSubject(makeMember(t, root, "b", 80, 100, 80, `"AI"`, 0))
	avg := teamAverages([]subject{a, b})
	if avg["influence"] != 70 {
		t.Fatalf("avg influence=%d want 70", avg["influence"])
	}
	if avg["contribution"] != 90 {
		t.Fatalf("avg contribution=%d want 90", avg["contribution"])
	}
	if a.Dev.User.get("language") != 60 {
		t.Fatalf("get language=%d", a.Dev.User.get("language"))
	}
}

func TestDisciplineCoverage(t *testing.T) {
	root := t.TempDir()
	a, _ := loadSubject(makeMember(t, root, "a", 60, 60, 60, `"AI","大数据"`, 0))
	b, _ := loadSubject(makeMember(t, root, "b", 60, 60, 60, `"AI","物联网"`, 0))
	shared, unique := disciplineCoverage([]subject{a, b})
	if _, ok := shared["AI"]; !ok {
		t.Fatalf("expected AI shared, got %v", shared)
	}
	found := map[string]bool{}
	for _, u := range unique {
		found[u] = true
	}
	if !found["大数据"] || !found["物联网"] {
		t.Fatalf("unique=%v", unique)
	}
}

func TestGradeAndActivitySum(t *testing.T) {
	cases := map[int]string{95: "卓越", 80: "优秀", 65: "良好", 45: "一般", 10: "较弱"}
	for v, want := range cases {
		if got := grade(v); got != want {
			t.Fatalf("grade(%d)=%s want %s", v, got, want)
		}
	}
	c, i, p := activitySum(activityData{CommitsCount: []int{1, 2}, IssuesCount: []int{0, 3}, PullRequestsCount: []int{1}})
	if c != 3 || i != 3 || p != 1 {
		t.Fatalf("activitySum=%d,%d,%d", c, i, p)
	}
}

func TestStrongestAndTopLanguages(t *testing.T) {
	root := t.TempDir()
	a, _ := loadSubject(makeMember(t, root, "weak", 50, 50, 50, `"AI"`, 0))
	b, _ := loadSubject(makeMember(t, root, "strong", 90, 90, 90, `"AI"`, 0))
	if strongest([]subject{a, b}).Login != "strong" {
		t.Fatal("expected strong to be strongest")
	}
	langs := topLanguages([]subject{a, b}, 5)
	if len(langs) == 0 || langs[0] != "Go" {
		t.Fatalf("top languages=%v", langs)
	}
}

func TestBuildMarkdownAndHTML(t *testing.T) {
	root := t.TempDir()
	a, _ := loadSubject(makeMember(t, root, "alice", 70, 95, 66, `"人工智能","大数据"`, 0))
	b, _ := loadSubject(makeMember(t, root, "bob", 60, 72, 77, `"人工智能","物联网"`, 9000))
	subs := []subject{a, b}

	md := buildInsightMarkdown("测试组", subs)
	for _, want := range []string{"科研团队洞察报告：测试组", "能力矩阵", "团队均值", "学科方向覆盖", "协作与分工建议", "含大量镜像"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}

	htmlOut := buildHTML("测试组", subs)
	for _, want := range []string{"<svg", "<polygon", "<line", "能力矩阵", "协作网络图", "课题组科研画像"} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("html missing %q", want)
		}
	}
}

func TestParseIntoUnwrapped(t *testing.T) {
	// tolerate payload without the {"data":..} envelope
	dir := t.TempDir()
	writeFixture(t, dir, "major.json", `{"categories":["X"]}`)
	var mj majorData
	if err := parseInto(filepath.Join(dir, "major.json"), &mj); err != nil {
		t.Fatalf("parseInto: %v", err)
	}
	if len(mj.Categories) != 1 || mj.Categories[0] != "X" {
		t.Fatalf("categories=%v", mj.Categories)
	}
}
