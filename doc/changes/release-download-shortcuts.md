# 新增 Release 资产下载能力

Release 命令组现在支持 `release +assets` 和 `release +download`，可以先查看某个发布包含的附件、zip/tar 源码包，再按附件 ID、附件文件名或源码包类型下载到本地目录。

下载链路新增了 client 层原始字节获取能力，并通过 `RuntimeContext.Download()` 暴露给 shortcut，避免 release 命令直接绕过统一客户端封装。附件 URL 支持 GitLink 返回的 `/api/attachments/<id>` 形式，源码包也支持完整 URL。

`release +download` 默认不覆盖已有文件；发布只有一个附件时可以直接下载，存在多个附件时要求显式传入 `--asset`，减少自动化脚本误下文件的风险。本次变更补充了 client、release shortcut、帮助文案、README 和 Skill reference 测试与文档。
