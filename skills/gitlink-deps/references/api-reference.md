# gitlink-deps API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能扫描依赖文件所依赖的接口与字段。全程只读。

## 采集的接口

### 列根目录（发现依赖文件）

```
GET /:owner/:repo/sub_entries.json?filepath=&ref={ref}
```

从 `entries[].name` 中匹配已知的依赖声明文件名。

### 读取依赖文件内容

```
GET /:owner/:repo/sub_entries.json?filepath={manifest}&ref={ref}
```

单文件查询时，`entries`（单对象）的 `content` 字段直接是**明文**文件内容。本技能据此读取 go.mod / package.json 等并解析。

> 注意：`raw/{path}` 接口对公开仓库可能返回 403，因此读取文件内容统一走 `sub_entries`。

## 识别的依赖文件

| 文件 | 生态 | 解析 |
|------|------|:----:|
| go.mod | Go | ✅ require 块 |
| package.json | Node.js | ✅ dependencies + devDependencies |
| requirements.txt | Python | ✅ 逐行 |
| Cargo.toml | Rust | ✅ [dependencies] |
| pom.xml | Java (Maven) | ✅ <dependency> |
| pyproject.toml / Pipfile / build.gradle / composer.json / Gemfile | 多语言 | 识别存在性 |

## 输出字段（JSON）

```json
{
  "owner": "...", "repo": "...",
  "ecosystems": ["Go"],
  "manifests": [{"file": "go.mod", "ecosystem": "Go", "count": 22}],
  "total_deps": 22, "direct_count": 7, "indirect_count": 15,
  "dependencies": [{"name": "...", "version": "...", "indirect": false, "manifest": "go.mod", "ecosystem": "Go"}],
  "risks": ["..."]
}
```

## 错误处理

沿用 gitlink-shared 错误码。依赖文件不存在或无法解析时跳过，不中断整体扫描。
