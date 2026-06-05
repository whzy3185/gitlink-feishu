# 依赖解析与风险规则

## 各生态解析规则

### Go（go.mod）

解析 `require (...)` 块与单行 `require`。识别 `module path vX.Y.Z` 形式，
带 `// indirect` 标记的归为间接依赖。

### Node.js（package.json）

解析 `dependencies`（直接）与 `devDependencies`（开发依赖，归为间接）。

### Python（requirements.txt）

逐行解析 `pkg==1.0` / `pkg>=1.0` / `pkg` 形式；跳过注释行与 `-e`、`-r` 等选项行。

### Rust（Cargo.toml）

解析 `[dependencies]` 段下的 `name = "version"` 与 `name = { version = "..." }`。

### Java（pom.xml）

正则提取 `<dependency>` 块的 `groupId:artifactId` 与 `version`。

## 风险评估规则

| 风险 | 触发条件 | 提示 |
|------|----------|------|
| 版本未锁定 | 依赖版本为 `*` / `latest` / 空，或以 `^` / `~` 开头 | 可能导致构建不可复现，建议锁定精确版本 |
| 直接依赖过多 | 直接依赖 > 50 | 建议定期审查，减少供应链攻击面 |
| 无依赖文件 | 未发现任何清单文件 | 可能是纯文档仓库，或依赖文件不在根目录 |

无风险命中时输出"未发现明显的依赖风险，依赖声明较为规范"。

## 直接 vs 间接依赖

- **直接依赖**：项目显式声明、直接使用的依赖。
- **间接依赖**：被直接依赖引入的传递依赖（go.mod 的 `// indirect`、package.json 的 `devDependencies` 在本工具中归类为非直接）。

区分二者有助于评估项目真正掌控的依赖规模。

## 局限

- 仅扫描仓库**根目录**的依赖文件；多模块 / monorepo 项目的子目录依赖需指定路径。
- 不解析锁文件（go.sum / package-lock.json）的完整依赖图，聚焦于声明文件中的直接意图。
