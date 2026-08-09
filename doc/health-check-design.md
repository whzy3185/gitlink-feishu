# 项目健康度诊断功能设计文档

## 概述

**方向B：项目健康度诊断** - 整合多个 GitLink API，一键诊断项目健康状况，提供评分和改进建议。

**核心价值**：体现 AI Agent 的"智能诊断"能力，不仅仅是 API 翻译，而是跨模块整合分析。

---

## CLI 命令设计

### 主命令

```bash
gitlink health diagnose owner/repo [flags]
```

### 参数

| 参数 | 短参数 | 说明 | 默认值 |
|------|--------|------|--------|
| `--format` | `-f` | 输出格式（text/json/markdown） | text |
| `--verbose` | `-v` | 显示详细信息 | false |
| `--db` | `-d` | SQLite 数据库路径（可选，用于历史对比） | - |

### 输出示例

```
🔍 项目健康度诊断报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📦 项目：gitlink-org/gitlink-cli
📊 总分：78/100 ⭐⭐⭐

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 诊断详情
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 文档完善度 (18/20)
   ├─ README 存在 ✓
   └─ README 内容丰富 ✓ (2,345 字符)

✅ 许可证合规性 (15/15)
   └─ 许可证: Apache-2.0 ✓

⚠️  社区活跃度 (16/25)
   ├─ 贡献者: 5 人 (建议: >10 人)
   ├─ Open Issues: 12 个
   └─ 最近 30 天 PR: 3 个 (建议: >5 个)

✅ 项目成熟度 (18/20)
   ├─ 分支: 3 个 ✓
   ├─ 标签: 8 个 ✓
   └─ Star: 156, Fork: 42 ✓

⚠️  CI/CD 配置 (11/20)
   └─ CI 构建记录: 有，但最近失败率较高 (33%)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 改进建议
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. 🤝 社区建设
   - 邀请更多开发者参与项目
   - 创建 CONTRIBUTING.md 指引贡献流程
   - 标记 "good first issue" 吸引新贡献者

2. 🔄 CI/CD 优化
   - 检查最近的构建失败原因
   - 添加自动化测试覆盖率报告
   - 配置代码质量检查工具

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📈 历史趋势（需要 --db 参数）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

2026-06-01: 72/100 ⬆️ +6
2026-06-15: 75/100 ⬆️ +3
2026-06-29: 78/100 ⬆️ +3

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 诊断维度与评分算法

### 1. 文档完善度 (20分)

**检查项**：
- README 是否存在（10分）
- README 内容长度（10分）
  - < 100 字符: 2分
  - 100-500 字符: 5分
  - 500-1500 字符: 8分
  - > 1500 字符: 10分

**API 调用**：
```
GET /api/{owner}/{repo}/readme.json
```

**实现逻辑**：
```go
func scoreReadme(readme map[string]interface{}) int {
    score := 0
    if readme == nil {
        return 0
    }
    score += 10 // README 存在
    
    content, _ := readme["content"].(string)
    if len(content) < 100 {
        score += 2
    } else if len(content) < 500 {
        score += 5
    } else if len(content) < 1500 {
        score += 8
    } else {
        score += 10
    }
    return score
}
```

---

### 2. 许可证合规性 (15分)

**检查项**：
- 是否有许可证（15分）

**API 调用**：
```
GET /api/{owner}/{repo}/detail.json
→ license_name 字段
```

**实现逻辑**：
```go
func scoreLicense(detail map[string]interface{}) int {
    licenseName, _ := detail["license_name"].(string)
    if licenseName != "" && licenseName != "null" {
        return 15
    }
    return 0
}
```

---

### 3. 社区活跃度 (25分)

**检查项**：
- 贡献者数量（10分）
  - 1-2 人: 3分
  - 3-5 人: 6分
  - 6-10 人: 8分
  - > 10 人: 10分

- 最近 PR/Issue 活跃度（15分）
  - 最近 30 天 PR 数量
  - Open Issue 处理情况

**API 调用**：
```
GET /api/{owner}/{repo}/contributors.json
GET /api/{owner}/{repo}/detail.json (issues_count, pull_requests_count)
```

**实现逻辑**：
```go
func scoreCommunity(contributors []interface{}, detail map[string]interface{}) int {
    score := 0
    
    // 贡献者评分
    contributorCount := len(contributors)
    if contributorCount <= 2 {
        score += 3
    } else if contributorCount <= 5 {
        score += 6
    } else if contributorCount <= 10 {
        score += 8
    } else {
        score += 10
    }
    
    // Issue/PR 活跃度（需要从 detail 获取）
    issuesCount, _ := detail["issues_count"].(float64)
    prCount, _ := detail["pull_requests_count"].(float64)
    
    // 简单评分逻辑
    if issuesCount > 0 || prCount > 0 {
        score += 10
    }
    if issuesCount > 10 || prCount > 5 {
        score += 5
    }
    
    return min(score, 25)
}
```

---

### 4. 项目成熟度 (20分)

**检查项**：
- 分支数量（5分）
  - 1 个: 2分
  - 2-3 个: 4分
  - > 3 个: 5分

- 标签数量（5分）
  - 0 个: 0分
  - 1-3 个: 3分
  - > 3 个: 5分

- Star/Fork 数量（10分）
  - Star < 10: 2分
  - Star 10-50: 5分
  - Star 50-200: 8分
  - Star > 200: 10分

**API 调用**：
```
GET /api/{owner}/{repo}/detail.json
→ branches_count, tags_count, praises_count, forked_count
```

**实现逻辑**：
```go
func scoreMaturity(detail map[string]interface{}) int {
    score := 0
    
    // 分支评分
    branchesCount, _ := detail["branches_count"].(float64)
    if branchesCount == 1 {
        score += 2
    } else if branchesCount <= 3 {
        score += 4
    } else {
        score += 5
    }
    
    // 标签评分
    tagsCount, _ := detail["tags_count"].(float64)
    if tagsCount == 0 {
        score += 0
    } else if tagsCount <= 3 {
        score += 3
    } else {
        score += 5
    }
    
    // Star 评分
    praisesCount, _ := detail["praises_count"].(float64)
    if praisesCount < 10 {
        score += 2
    } else if praisesCount < 50 {
        score += 5
    } else if praisesCount < 200 {
        score += 8
    } else {
        score += 10
    }
    
    return score
}
```

---

### 5. CI/CD 配置 (20分)

**检查项**：
- 是否有 CI 构建记录（10分）
- 最近构建成功率（10分）
  - 无记录: 0分
  - 成功率 < 50%: 3分
  - 成功率 50-80%: 6分
  - 成功率 > 80%: 10分

**API 调用**：
```
GET /api/v1/{owner}/{repo}/builds.json
```

**实现逻辑**：
```go
func scoreCI(builds []interface{}) int {
    if len(builds) == 0 {
        return 0
    }
    
    score := 10 // 有 CI 记录
    
    // 计算成功率
    successCount := 0
    for _, build := range builds {
        if b, ok := build.(map[string]interface{}); ok {
            if status, _ := b["status"].(string); status == "success" {
                successCount++
            }
        }
    }
    
    successRate := float64(successCount) / float64(len(builds))
    if successRate < 0.5 {
        score += 3
    } else if successRate < 0.8 {
        score += 6
    } else {
        score += 10
    }
    
    return score
}
```

---

## 技术实现方案

### 方案选择：扩展现有 health.go 模块

**理由**：
1. health.go 已有数据采集基础设施（SQLite、rate limiter）
2. 可以复用现有的 API 调用逻辑
3. 保持代码组织一致性

### 文件结构

```
shortcuts/health/
├── health.go          # 现有 fetch 命令
├── diagnose.go        # 新增 diagnose 命令
├── scoring.go         # 评分算法
├── suggestions.go     # 改进建议生成
└── health_test.go     # 单元测试
```

### 核心代码结构

```go
// diagnose.go
package health

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
    return []*common.Shortcut{
        // 现有 fetch 命令...
        {
            Name:        "diagnose",
            Description: "Diagnose project health and provide improvement suggestions",
            Flags: []common.Flag{
                {Name: "format", Short: "f", Usage: "Output format (text/json/markdown)", Default: "text"},
                {Name: "verbose", Short: "v", Usage: "Show detailed information"},
                {Name: "db", Short: "d", Usage: "SQLite database path for historical comparison"},
            },
            Run: func(ctx *common.RuntimeContext) error {
                return runDiagnose(ctx)
            },
        },
    }
}

func runDiagnose(ctx *common.RuntimeContext) error {
    // 1. 解析 owner/repo
    if err := ctx.ResolveOwnerRepo(); err != nil {
        return err
    }
    
    // 2. 并发获取数据
    data, err := fetchDiagnosisData(ctx)
    if err != nil {
        return err
    }
    
    // 3. 计算评分
    report := calculateHealthScore(data)
    
    // 4. 生成改进建议
    report.Suggestions = generateSuggestions(report)
    
    // 5. 输出结果
    return outputReport(ctx, report)
}
```

---

## 需要调用的 API 列表

| API | 用途 | 诊断维度 |
|-----|------|----------|
| `GET /api/{owner}/{repo}/detail.json` | 项目详情 | 许可证、成熟度、社区 |
| `GET /api/{owner}/{repo}/readme.json` | README 内容 | 文档完善度 |
| `GET /api/{owner}/{repo}/contributors.json` | 贡献者列表 | 社区活跃度 |
| `GET /api/v1/{owner}/{repo}/builds.json` | CI 构建记录 | CI/CD 配置 |

---

## i18n 翻译支持

### zh-CN.json

```json
{
  "cmd.health.diagnose.short": "诊断项目健康度并提供改进建议",
  "cmd.health.diagnose.format": "输出格式 (text/json/markdown)",
  "cmd.health.diagnose.verbose": "显示详细信息",
  "health.score.documentation": "文档完善度",
  "health.score.license": "许可证合规性",
  "health.score.community": "社区活跃度",
  "health.score.maturity": "项目成熟度",
  "health.score.ci": "CI/CD 配置",
  "health.suggestion.community": "社区建设",
  "health.suggestion.ci": "CI/CD 优化"
}
```

### en-US.json

```json
{
  "cmd.health.diagnose.short": "Diagnose project health and provide improvement suggestions",
  "cmd.health.diagnose.format": "Output format (text/json/markdown)",
  "cmd.health.diagnose.verbose": "Show detailed information",
  "health.score.documentation": "Documentation Quality",
  "health.score.license": "License Compliance",
  "health.score.community": "Community Activity",
  "health.score.maturity": "Project Maturity",
  "health.score.ci": "CI/CD Configuration",
  "health.suggestion.community": "Community Building",
  "health.suggestion.ci": "CI/CD Optimization"
}
```

---

## 开发计划

### 阶段 1：核心功能（2-3 天）
- [ ] 创建 diagnose.go 文件
- [ ] 实现数据获取逻辑（并发调用 API）
- [ ] 实现评分算法（scoring.go）
- [ ] 实现 text 格式输出

### 阶段 2：增强功能（1-2 天）
- [ ] 实现改进建议生成（suggestions.go）
- [ ] 实现 json/markdown 格式输出
- [ ] 添加历史对比功能（需要 --db 参数）

### 阶段 3：测试与文档（1 天）
- [ ] 编写单元测试
- [ ] 添加 i18n 翻译
- [ ] 更新 README 和 Skills 文档
- [ ] 注册到 register.go

---

## 创新点总结

1. **跨模块整合**：整合 repo、issue、ci 等多个模块的 API
2. **智能评分**：基于多维度算法计算健康度分数
3. **改进建议**：根据诊断结果提供可操作的改进建议
4. **历史对比**：支持历史数据对比，追踪项目健康度变化
5. **多格式输出**：支持 text/json/markdown 格式，方便集成到其他工具

---

## 与其他方向对比

| 方向 | 创新性 | 技术难度 | 实用性 | AI Agent 价值 |
|------|--------|----------|--------|---------------|
| A. 开发者画像 | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **B. 项目健康度诊断** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| C. 智能工作流编排 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| D. 交互式反馈收集 | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐ |

**方向B 优势**：
- 技术难度适中，可在比赛时间内完成
- 实用性强，对开源项目维护者有实际价值
- 体现 AI Agent 的整合分析能力
- 可扩展性好，未来可添加更多诊断维度
