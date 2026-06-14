# gatekeeper 仓库体检报告 —— Gitlink/gitlink-cli（2026-06-10）

> 对 **113 个 open PR** 全量 dry-run（**只读，零写入**）· 策略 `gatekeeper.yaml` · 成功 113 / 失败 0
>
> **诚实口径**：批扫未注入 AI 审查发现，review_findings 维按 0 发现计满分（**该维度未评**）；CI 维按 `--skip-ci` 统一记 unknown（半分）。其余维度为真实采集。因此**总分代表「除人工/AI 审查外的工程卫生分」，偏乐观**；裁决分布同理。

## 总览

- 裁决分布：COMMENT **6** · PASS **105** · REQUEST_CHANGES **2**
- 分数：min 70 / 中位 90 / 均值 88.5 / max 95
- **0%** 的 PR 测试覆盖维 0 分（改动不带任何测试）
- **96%** 的 PR 未关联 issue
- **2%** 的 PR 触发 REQUEST_CHANGES（硬门禁或低分）

硬门禁命中：`require_tests_for_src_changes` × 2

## 全量明细（按分数降序）

| PR | 标题 | 作者 | 总分 | 裁决 | 硬门禁失败 | 卫生(描述/关联/体量) |
|----|------|------|-----:|------|-----------|---------------------|
| [#145](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/145) | fix(issue): preserve metadata during batch close | dtwdtw | 95 | PASS | — | ✓/✓/✓ |
| [#218](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/218) | feat(skills): 新增 科研Fork影响力分析 的skill : gitlink-re | yangsai | 90 | PASS | — | ✓/✗/✓ |
| [#177](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/177) | feat(wiki): add wiki management shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#217](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/217) | feat(commands): add command catalog export | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#216](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/216) | feat(api): support saved variables in batch plan | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#214](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/214) | feat(pr): add conversation comment shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#213](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/213) | feat(repo): add mirror sync shortcut | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#212](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/212) | feat(feedback): add feedback shortcut | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#211](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/211) | feat(repo): add profile view shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#210](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/210) | feat(skills): 新增维护者交接与分支治理 Skills | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#208](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/208) | feat(user): add pinned project shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#207](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/207) | feat(user): add statistics shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#206](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/206) | feat(commit): add commit inspection shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#204](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/204) | feat(org): 增强组织团队与成员管理快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#203](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/203) | feat(user): 增加用户画像分析快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#202](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/202) | feat(ignore): add ignore template shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#201](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/201) | feat(account): add account auth shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#200](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/200) | feat(pr): add review journal shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#199](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/199) | feat(code): add read-only code browsing shortcut | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#198](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/198) | feat(message): 增加消息中心快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#197](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/197) | feat(message-settings): 增加消息通知设置快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#194](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/194) | fix(pr): 补齐 pr +view 的合并与关闭时间字段 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#193](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/193) | feat(shortcut): add shortcuts/wiki | co63oc | 90 | PASS | — | ✓/✗/✓ |
| [#192](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/192) | feat(repo): add navigation unit shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#191](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/191) | feat(user): add profile shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#187](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/187) | feat(org): add team project bulk shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#186](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/186) | feat(ref): add branch and tag shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#185](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/185) | Add workflow pull request review queue | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#184](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/184) | Add workflow release notes generator | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#183](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/183) | feat(project): add lifecycle flow shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#182](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/182) | feat(issue): add journal maintenance shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#181](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/181) | feat(topic): add project topic shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#180](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/180) | feat(template): add project template shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#179](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/179) | feat(dataset): add research dataset shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#178](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/178) | feat(contents): add repository content shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#176](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/176) | feat(user): add dashboard shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#175](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/175) | feat(notification): add message and setting shor | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#174](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/174) | feat(public-key): add SSH key shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#173](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/173) | feat(account): add cancellation shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#172](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/172) | feat(account): add security shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#171](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/171) | feat(oauth): add token shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#170](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/170) | Add repository file search and batch commit shor | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#167](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/167) | feat(account): add email verification shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#164](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/164) | Add pull request review comment management short | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#163](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/163) | Add complete issue comment management shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#160](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/160) | Add GitLink feedback submission shortcut | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#158](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/158) | Add code trace analysis shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#153](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/153) | feat(shortcut): add shortcuts/ignore | co63oc | 90 | PASS | — | ✓/✗/✓ |
| [#151](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/151) | feat(transfer): add transfer request shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#135](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/135) | feat(dev): add developer resource shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#118](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/118) | feat(access): add project access shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#114](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/114) | feat(mirror): add mirror repository shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#113](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/113) | feat(todo): add request approval shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#107](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/107) | feat(star): add starred project shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#83](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/83) | feat(org): add team project binding shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#82](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/82) | feat(meta): add attachment and metadata shortcut | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#78](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/78) | feat(branch): complete OpenAPI shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#76](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/76) | feat(notification): add OpenAPI shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#72](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/72) | feat(template): add project template shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#70](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/70) | feat(user): add account and stats shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#65](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/65) | feat(wiki): add OpenAPI shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#64](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/64) | feat(dataset): add OpenAPI shortcuts | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#63](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/63) | feat(code): add repository code OpenAPI shortcut | wangyue111 | 90 | PASS | — | ✓/✗/✓ |
| [#152](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/152) | chore(doc): fix README.md | co63oc | 90 | PASS | — | ✓/✗/✓ |
| [#137](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/137) | feat(skills): 增强 7 个 Agent Skill + 新增 2 个 Skill（ | whale | 90 | PASS | — | ✓/✗/✓ |
| [#149](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/149) | feat(skills): 新增 学者/团队科研画像生成 的skill : gitlink-sc | yangsai | 90 | PASS | — | ✓/✗/✓ |
| [#148](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/148) | feat(skills): 新增 科研热点追踪与知识图谱构建 的skill : gitlink- | yangsai | 90 | PASS | — | ✓/✗/✓ |
| [#144](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/144) | feat(skills): 新增 3 个 Agent Skill — wiki-builder, | whale | 90 | PASS | — | ✓/✗/✓ |
| [#134](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/134) | 新增 shell 自动补全命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#99](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/99) | 新增 5 个仓库检查快捷命令 (languages/contributors/files/tag | jiangtx | 90 | PASS | — | ✓/✗/✓ |
| [#86](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/86) | fix: preserve issue metadata on update | dtwdtw | 90 | PASS | — | ✓/✗/✓ |
| [#73](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/73) | feat(user): add SSH key shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#67](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/67) | feat(repo): add repository units shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#60](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/60) | feat: add notification shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#58](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/58) | feat: add repository reaction shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#126](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/126) | feat(skills): 新增 gitlink-scaffold 社区健康文件体检 Skill | Ct201314 | 90 | PASS | — | ✓/✗/✓ |
| [#56](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/56) | feat: add git tag shortcut group | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#125](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/125) | feat(skills): 新增 gitlink-newcomer 新人引导 Skill | Ct201314 | 90 | PASS | — | ✓/✗/✓ |
| [#127](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/127) | feat(skills): 新增 gitlink-deps 依赖追踪 Skill | Ct201314 | 90 | PASS | — | ✓/✗/✓ |
| [#128](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/128) | feat(skills): 新增 gitlink-contributor 贡献者致谢与成长 Sk | Ct201314 | 90 | PASS | — | ✓/✗/✓ |
| [#129](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/129) |  feat(skills): 新增 gitlink-kb 知识库问答 Skill | Ct201314 | 90 | PASS | — | ✓/✗/✓ |
| [#115](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/115) | feat: add catalog template shortcuts | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#116](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/116) | 新增仓库洞察快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#119](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/119) | 新增仓库转移快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#122](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/122) | 完善仓库 README 快捷命令 | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#50](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/50) | feat: add wiki shortcut group | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#54](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/54) | gitlink-growth 开源贡献者成长系统 Skill 贡献 | yingjie | 90 | PASS | — | ✓/✗/✓ |
| [#23](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/23) | feat: support fork metadata in pr create | Mengz | 90 | PASS | — | ✓/✗/✓ |
| [#196](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/196) | feat(release): 增加发布资产管理快捷命令 | Mengz | 88 | PASS | — | ✓/✗/✓ |
| [#215](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/215) | fix(client): improve API robustness | wangyue111 | 87 | PASS | — | ✓/✗/✓ |
| [#147](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/147) | feat(shortcuts): 新增 wiki/commit/file/star/watch  | chroe | 86 | PASS | — | ✓/✗/✓ |
| [#209](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/209) | feat(milestone): 增加里程碑进度分析快捷命令 | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#205](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/205) | fix(issue): 修复详情缺失并保护更新元数据 | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#195](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/195) | feat(compare): 新增 compare 汇总与提交筛选能力 | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#190](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/190) | Add workflow release readiness gate | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#189](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/189) | Add workflow duplicate issue detection | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#188](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/188) | Add workflow dependency risk audit | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#165](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/165) | feat(issue): add batch maintenance shortcuts | wangyue111 | 85 | PASS | — | ✓/✗/✓ |
| [#159](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/159) | Add member application workflow shortcuts | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#77](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/77) | feat(journal): add issue and PR comment shortcut | wangyue111 | 85 | PASS | — | ✓/✗/✓ |
| [#150](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/150) | 新增 Issue 批量导出命令 | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#142](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/142) | 新增 PR 本地检出命令 | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#100](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/100) | 查看指定时间范围的开发统计 | jiangtx | 85 | PASS | — | ✗/✗/✓ |
| [#101](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/101) | 查看用户项目动态 | jiangtx | 85 | PASS | — | ✗/✗/✓ |
| [#21](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/21) | feat: add attachment shortcut group | Mengz | 85 | PASS | — | ✓/✗/✓ |
| [#139](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/139) | feat(wiki): 新增 Wiki 页面与目录管理 Shortcuts | whale | 82 | COMMENT | — | ✓/✗/✓ |
| [#130](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/130) | feat(workflows): 新增 gitlink-flow 社区运营自动化端到端工作流 | Ct201314 | 82 | COMMENT | — | ✓/✗/✓ |
| [#97](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/97) | 基础设施修复 | jiangtx | 81 | COMMENT | — | ✗/✗/✓ |
| [#123](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/123) | 新增 Release 资产下载命令 | Mengz | 80 | COMMENT | — | ✗/✗/✓ |
| [#103](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/103) | feat: 新建 pm 模块，添加 6 条项目管理命令 | wyxttn | 78 | COMMENT | — | — |
| [#131](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/131) | 子赛题三 - Java-Gatekeeper 端到端自动化质量门禁工作流 | xxxx12 | 75 | REQUEST_CHANGES | require_tests_for_src_changes | ✓/✓/✓ |
| [#30](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/30) | 增加wiki管理的shortcut | camelliamc | 74 | COMMENT | — | — |
| [#146](https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/146) | feat: 新增 Showcase Dashboard 交互式展示页 | chroe | 70 | REQUEST_CHANGES | require_tests_for_src_changes | ✓/✗/✓ |

## 这份报告说明了什么

- 同一份 `gatekeeper.yaml` 策略可以**无人值守地体检一个真实活跃仓库的全部积压**——确定性评分意味着大规模治理零 AI 成本，AI 只在需要语义判断（review_findings）时按需介入。
- 任何人重跑本报告（`python3 scripts/gatekeeper_sweep.py`）会对同一组 PR 得到同样的分数与裁决。
