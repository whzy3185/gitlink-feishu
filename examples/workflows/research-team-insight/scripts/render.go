package main

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
)

var palette = []string{"#2f6feb", "#1f883d", "#bf8700", "#cf222e", "#6f42c1", "#0969da"}

// ---------- Markdown insight ----------

func buildInsightMarkdown(team string, subs []subject) string {
	var b strings.Builder
	n := len(subs)
	avg := teamAverages(subs)

	fmt.Fprintf(&b, "# 🔬 科研团队洞察报告：%s\n\n", team)
	fmt.Fprintf(&b, "> 成员 %d 人 ｜ 数据来源：gitlink-cli（只读）\n\n", n)

	b.WriteString("## 一、能力矩阵（0–100）\n\n")
	b.WriteString("| 成员 |")
	for _, d := range abilityDims {
		fmt.Fprintf(&b, " %s |", d.ZH)
	}
	b.WriteString("\n|------|")
	for range abilityDims {
		b.WriteString("------|")
	}
	b.WriteString("\n")
	for _, s := range subs {
		fmt.Fprintf(&b, "| %s（@%s） |", s.displayName(), s.Login)
		for _, d := range abilityDims {
			fmt.Fprintf(&b, " %d |", s.Dev.User.get(d.Key))
		}
		b.WriteString("\n")
	}
	b.WriteString("| **团队均值** |")
	for _, d := range abilityDims {
		fmt.Fprintf(&b, " **%d** |", avg[d.Key])
	}
	b.WriteString("\n\n")

	// strongest / weakest dimension
	type dimAvg struct {
		zh string
		v  int
	}
	var das []dimAvg
	for _, d := range abilityDims {
		das = append(das, dimAvg{d.ZH, avg[d.Key]})
	}
	sort.Slice(das, func(i, j int) bool { return das[i].v > das[j].v })
	fmt.Fprintf(&b, "- 团队最强项：**%s**（%d）；短板：**%s**（%d，%s）\n\n",
		das[0].zh, das[0].v, das[len(das)-1].zh, das[len(das)-1].v, grade(das[len(das)-1].v))

	// disciplines
	shared, unique := disciplineCoverage(subs)
	b.WriteString("## 二、学科方向覆盖\n\n")
	fmt.Fprintf(&b, "- 覆盖方向总数：**%d**\n", len(shared)+len(unique))
	b.WriteString("- 共有方向（团队共识）：" + sharedSummary(shared, 10) + "\n")
	b.WriteString("- 独有方向（个人特长）：" + joinLimit(unique, 15) + "\n\n")

	// languages
	b.WriteString("## 三、技术栈覆盖\n\n")
	langs := topLanguages(subs, 8)
	b.WriteString("- 团队主力语言：" + emptyOr(strings.Join(langs, ", "), "无") + "\n\n")

	// roles
	b.WriteString("## 四、角色结构\n\n")
	for _, s := range subs {
		owner := s.Role.Role["owner"].Count
		dev := s.Role.Role["developer"].Count
		note := ""
		if s.Info.MirrorProjectsCount > 100 {
			note = "　⚠️ 含大量镜像，分布偏泛"
		}
		fmt.Fprintf(&b, "- %s：参与 %s 个项目（Owner %d / Developer %d）%s\n",
			s.displayName(), dashIfZero(s.Role.TotalProjectsCount), owner, dev, note)
	}
	b.WriteString("\n")

	// suggestions
	b.WriteString("## 五、协作与分工建议\n\n")
	lead := strongest(subs)
	fmt.Fprintf(&b, "- **核心建议**：%s（@%s）综合能力最强，适合作为方向负责人。\n", lead.displayName(), lead.Login)
	if top := topShared(shared); top != "" {
		fmt.Fprintf(&b, "- **团队聚焦**：在共有方向「%s」上成员重叠度最高，可作为协作主线。\n", top)
	}
	if len(unique) > 0 {
		fmt.Fprintf(&b, "- **互补利用**：%s 等为个人独有方向，可作为跨学科延展点。\n", joinLimit(unique, 5))
	}
	fmt.Fprintf(&b, "- **能力补强**：团队短板为「%s」，建议引入该维度更强的协作者或专项提升。\n\n", das[len(das)-1].zh)

	b.WriteString("---\n\n*本报告由 research-team-insight 工作流自动生成，链路：search/profile → 聚合 → 可视化 → 归档。配套可视化见同目录 `team-report.html`。*\n")
	return b.String()
}

// ---------- HTML report ----------

func buildHTML(team string, subs []subject) string {
	avg := teamAverages(subs)
	shared, unique := disciplineCoverage(subs)

	var matrix strings.Builder
	matrix.WriteString("<tr><th>成员</th>")
	for _, d := range abilityDims {
		fmt.Fprintf(&matrix, "<th>%s</th>", d.ZH)
	}
	matrix.WriteString("</tr>")
	for i, s := range subs {
		color := palette[i%len(palette)]
		fmt.Fprintf(&matrix, `<tr><td><span class="dot" style="background:%s"></span>%s <span class="muted">@%s</span></td>`,
			color, html.EscapeString(s.displayName()), html.EscapeString(s.Login))
		for _, d := range abilityDims {
			fmt.Fprintf(&matrix, "<td>%d</td>", s.Dev.User.get(d.Key))
		}
		matrix.WriteString("</tr>")
	}
	matrix.WriteString("<tr><td><b>团队均值</b></td>")
	for _, d := range abilityDims {
		fmt.Fprintf(&matrix, "<td><b>%d</b></td>", avg[d.Key])
	}
	matrix.WriteString("</tr>")

	sharedTags := tagsHTML(sharedTagList(shared))
	uniqueTags := tagsHTML(unique)

	body := fmt.Sprintf(`
<h1>🔬 课题组科研画像：%s</h1>
<div class="sub">成员 %d 人 · 数据来源 gitlink-cli（只读）</div>
<div class="grid">
  <div class="card"><h2>开发能力雷达</h2>%s%s</div>
  <div class="card"><h2>能力矩阵（0–100）</h2><table>%s</table></div>
</div>
<div class="grid">
  <div class="card"><h2>共有研究方向</h2>%s</div>
  <div class="card"><h2>独有研究方向</h2>%s</div>
</div>
<div class="card"><h2>协作网络图（连线=共同方向数）</h2>%s</div>
`,
		html.EscapeString(team), len(subs),
		radarSVG(subs), radarLegend(subs),
		matrix.String(), sharedTags, uniqueTags, networkSVG(subs))

	return page(fmt.Sprintf("课题组科研画像 - %s", team), body)
}

func page(title, body string) string {
	css := `body{font-family:-apple-system,'Segoe UI','Microsoft YaHei',sans-serif;margin:0;background:#f6f8fa;color:#24292f}
.wrap{max-width:980px;margin:0 auto;padding:24px}
h1{font-size:22px;margin:0 0 4px}h2{font-size:16px;margin:18px 0 10px;border-left:4px solid #2f6feb;padding-left:8px}
.sub{color:#57606a;font-size:13px;margin-bottom:16px}
.card{background:#fff;border:1px solid #e6e8eb;border-radius:10px;padding:16px;margin-bottom:16px}
.grid{display:flex;gap:16px;flex-wrap:wrap}.grid>.card{flex:1;min-width:300px}
.tag{display:inline-block;background:#eaf1ff;color:#2f6feb;border-radius:12px;padding:3px 10px;margin:3px;font-size:12px}
.muted{color:#8b949e;font-size:12px}.legend{font-size:12px;color:#57606a;margin-top:8px}
table{border-collapse:collapse;width:100%}td,th{border:1px solid #eaecef;padding:6px 8px;text-align:left;font-size:13px}th{background:#f6f8fa}
.dot{display:inline-block;width:10px;height:10px;border-radius:50%;margin-right:6px;vertical-align:middle}`
	return fmt.Sprintf(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`+
		`<meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>%s</style></head>`+
		`<body><div class="wrap">%s<p class="muted" style="margin-top:24px">由 research-team-insight 工作流生成 · gitlink-cli profile（只读）</p></div></body></html>`,
		html.EscapeString(title), css, body)
}

// ---------- SVG ----------

func radarSVG(subs []subject) string {
	const size = 340.0
	cx, cy := size/2, size/2
	r := size/2 - 50
	n := len(abilityDims)
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f" xmlns="http://www.w3.org/2000/svg">`, size, size, size, size)
	for k := 1; k <= 4; k++ {
		fmt.Fprintf(&sb, `<circle cx="%.0f" cy="%.0f" r="%.1f" fill="none" stroke="#e6e8eb"/>`, cx, cy, r*float64(k)/4)
	}
	for i, d := range abilityDims {
		ang := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
		x, y := cx+r*math.Cos(ang), cy+r*math.Sin(ang)
		fmt.Fprintf(&sb, `<line x1="%.0f" y1="%.0f" x2="%.1f" y2="%.1f" stroke="#e6e8eb"/>`, cx, cy, x, y)
		lx, ly := cx+(r+20)*math.Cos(ang), cy+(r+20)*math.Sin(ang)
		anchor := "middle"
		if math.Cos(ang) > 0.3 {
			anchor = "start"
		} else if math.Cos(ang) < -0.3 {
			anchor = "end"
		}
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="%s" font-size="12" fill="#444">%s</text>`, lx, ly+4, anchor, d.ZH)
	}
	for i, s := range subs {
		color := palette[i%len(palette)]
		var pts strings.Builder
		for j, d := range abilityDims {
			ang := -math.Pi/2 + 2*math.Pi*float64(j)/float64(n)
			rr := r * float64(clamp(s.Dev.User.get(d.Key), 0, 100)) / 100
			if j > 0 {
				pts.WriteString(" ")
			}
			fmt.Fprintf(&pts, "%.1f,%.1f", cx+rr*math.Cos(ang), cy+rr*math.Sin(ang))
		}
		fmt.Fprintf(&sb, `<polygon points="%s" fill="%s22" stroke="%s" stroke-width="2"/>`, pts.String(), color, color)
	}
	sb.WriteString("</svg>")
	return sb.String()
}

func radarLegend(subs []subject) string {
	var sb strings.Builder
	sb.WriteString(`<div class="legend">`)
	for i, s := range subs {
		color := palette[i%len(palette)]
		fmt.Fprintf(&sb, `<span class="dot" style="background:%s"></span>%s&nbsp;&nbsp;`, color, html.EscapeString(s.displayName()))
	}
	sb.WriteString("</div>")
	return sb.String()
}

func networkSVG(subs []subject) string {
	n := len(subs)
	const size = 420.0
	if n < 2 {
		return `<p class="muted">成员不足 2 人，无协作网络。</p>`
	}
	cx, cy := size/2, size/2
	r := size/2 - 64
	type pt struct{ x, y float64 }
	pos := make([]pt, n)
	for i := range subs {
		ang := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
		pos[i] = pt{cx + r*math.Cos(ang), cy + r*math.Sin(ang)}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f" xmlns="http://www.w3.org/2000/svg">`, size, size, size, size)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			shared := sharedCount(subs[i].Major, subs[j].Major)
			if shared > 0 {
				w := math.Min(6, 1+float64(shared)/3)
				fmt.Fprintf(&sb, `<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="#9ab4e8" stroke-width="%.1f" opacity="0.75"><title>%s ↔ %s: %d 个共同方向</title></line>`,
					pos[i].x, pos[i].y, pos[j].x, pos[j].y, w, html.EscapeString(subs[i].Login), html.EscapeString(subs[j].Login), shared)
			}
		}
	}
	for i, s := range subs {
		color := palette[i%len(palette)]
		label := []rune(s.displayName())
		if len(label) > 4 {
			label = label[:4]
		}
		fmt.Fprintf(&sb, `<circle cx="%.0f" cy="%.0f" r="22" fill="%s"/><text x="%.0f" y="%.0f" text-anchor="middle" font-size="11" fill="#fff">%s</text>`,
			pos[i].x, pos[i].y, color, pos[i].x, pos[i].y+4, html.EscapeString(string(label)))
	}
	sb.WriteString("</svg>")
	return sb.String()
}

// ---------- small helpers ----------

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sharedCount(a, b []string) int {
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	c := 0
	for _, y := range b {
		if set[y] {
			c++
		}
	}
	return c
}

func tagsHTML(items []string) string {
	if len(items) == 0 {
		return `<span class="muted">无</span>`
	}
	var sb strings.Builder
	for _, it := range items {
		fmt.Fprintf(&sb, `<span class="tag">%s</span>`, html.EscapeString(it))
	}
	return sb.String()
}

func sharedTagList(shared map[string][]string) []string {
	type kv struct {
		k string
		n int
	}
	var list []kv
	for c, who := range shared {
		list = append(list, kv{c, len(who)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	var out []string
	for _, e := range list {
		out = append(out, fmt.Sprintf("%s ×%d", e.k, e.n))
	}
	return out
}

func sharedSummary(shared map[string][]string, limit int) string {
	list := sharedTagList(shared)
	return emptyOr(joinLimit(list, limit), "无")
}

func topShared(shared map[string][]string) string {
	best, bestN := "", 0
	for c, who := range shared {
		if len(who) > bestN {
			bestN, best = len(who), c
		}
	}
	return best
}

func joinLimit(items []string, limit int) string {
	if len(items) > limit {
		items = items[:limit]
	}
	return strings.Join(items, ", ")
}

func emptyOr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func dashIfZero(n int) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}
