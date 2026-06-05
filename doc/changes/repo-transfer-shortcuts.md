# 仓库转移 Shortcut

本次变更为 `repo` 命令组补齐 GitLink 项目转移发起端能力：

- `repo +transfer-orgs`：列出当前仓库可转移到的组织。
- `repo +transfer`：向目标用户或组织发起仓库转移申请。
- `repo +transfer-cancel`：取消待处理的仓库转移申请。

这组命令覆盖仓库所有者侧的转移流程，和用户待办审批侧的接受/拒绝转移申请互补。维护者在迁移项目归属、把个人仓库移交给组织、或由 Agent 编排仓库治理流程时，不再需要手写 Raw API。

`repo +transfer` 和 `repo +transfer-cancel` 都支持 `--dry-run`，会输出将要请求的 method、path 和 payload，不修改线上数据，便于在执行高风险操作前确认目标仓库和目标所有者。

单元测试覆盖了 API method/path、转移 payload、缺少目标所有者校验，以及 dry-run 不触发远端请求；README、中文 README 和 `gitlink-repo` Skill 已同步补充示例。
