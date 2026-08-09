# 新增重复 Issue 候选检测工作流

`gitlink-cli workflow +issue-dedupe` 新增了只读的重复 Issue 候选检测能力。命令可以读取本地 Issue JSON，也可以通过 GitLink API 只读拉取指定仓库的 Issue 列表，再基于标题、正文和标签分词计算相似度，输出候选重复对、置信度、共享关键词和处理建议。

该功能面向 Issue 较多的仓库维护场景：维护者可以先处理高置信重复候选，把后创建的 Issue 链接到原始问题，减少重复排查成本；也可以通过 `--threshold` 调整相似度阈值，通过 `--max-pairs` 控制输出规模。输出支持 `table`、`markdown` 和 `json`，适合终端查看、Issue 评论草稿或自动化脚本消费。

命令保持安全边界，不会自动关闭、评论或修改 Issue。测试覆盖了本地 JSON 输入、状态和数量过滤、远端只读查询参数、相似度排序、中文 markdown 渲染、table 输出和 shortcut 注册。
