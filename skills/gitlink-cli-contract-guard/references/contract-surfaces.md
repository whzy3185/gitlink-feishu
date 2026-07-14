# CLI 契约面映射

这个 skill 的第一步不是直接评论代码，而是先判断改动触到了哪类契约面。

## 1. 参数契约

典型文件：

- `cmd/root.go`
- `cmd/*/*.go`
- `shortcuts/*/*.go`
- `shortcuts/common/runner.go`
- `shortcuts/common/types.go`

重点观察：

- `Name`
- `Short`
- `Default`
- `Required`
- `Bool`
- `Usage`

高风险信号：

- 旧 flag 被重命名或移除
- 必填/可选规则变化
- 默认值变化但没有说明
- 示例命令仍使用旧 flag

## 2. 帮助契约

典型文件：

- `cmd/root.go`
- `cmd/root_test.go`
- `shortcuts/register.go`
- `internal/i18n/**`
- `README.md`
- `README.zh-CN.md`

重点观察：

- 命令分组和层级
- `Use` / `Short` / `Long`
- 中英文帮助文案
- `--help` 输出快照或断言

高风险信号：

- 命令已改，但帮助仍描述旧行为
- 中文或英文帮助不一致
- i18n key 缺失导致回退或错误

## 3. 输出契约

典型文件：

- `shortcuts/common/types.go`
- `internal/output/**`
- `shortcuts/*/*_test.go`
- `cmd/*/*_test.go`

重点观察：

- `--format json`
- envelope 结构
- 字段名和字段类型
- 是否仍适合脚本消费

高风险信号：

- 字段重命名
- 类型从字符串改为对象/数组
- 原有必备字段消失
- 错误输出结构不再稳定

## 4. 错误契约

典型文件：

- `internal/client/**`
- `internal/auth/**`
- `cmd/**`
- `internal/i18n/**`

重点观察：

- `suggestFix`
- 缺参错误
- 校验错误
- header/path/query 渲染后行为
- 中文提示编码

高风险信号：

- 用户可见中文乱码
- 错误变得不可操作
- 渲染后非法 header/path 没有被拦截

## 5. 文档契约

典型文件：

- `README.md`
- `README.zh-CN.md`
- `examples/**`
- 命令帮助中的示例

高风险信号：

- 文档示例已不能运行
- 示例使用了不存在的参数
- 文档与实际默认值不一致
- 文档新增内容出现乱码
