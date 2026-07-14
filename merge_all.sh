#!/bin/bash
set -e

# 添加所有 fork remote
git remote add lijiabin1234 https://www.gitlink.org.cn/Lijiabin1234/gitlink-cli.git 2>/dev/null || true
git remote add maxwell7822 https://www.gitlink.org.cn/Maxwell7822/gitlink-cli.git 2>/dev/null || true
git remote add mengz https://www.gitlink.org.cn/Mengz/gitlink-cli.git 2>/dev/null || true
git remote add surponess https://www.gitlink.org.cn/Surponess/gitlink-cli.git 2>/dev/null || true
git remote add taoyouce https://www.gitlink.org.cn/Taoyouce/gitlink-cli.git 2>/dev/null || true
git remote add chroe https://www.gitlink.org.cn/chroe/gitlink-cli.git 2>/dev/null || true
git remote add co63oc https://www.gitlink.org.cn/co63oc/gitlink-cli.git 2>/dev/null || true
git remote add dtwdtw https://www.gitlink.org.cn/dtwdtw/gitlink-cli.git 2>/dev/null || true
git remote add heke1228 https://www.gitlink.org.cn/heke1228/gitlink-cli.git 2>/dev/null || true
git remote add jiangtx https://www.gitlink.org.cn/jiangtx/gitlink-cli.git 2>/dev/null || true
git remote add luwanzhou https://www.gitlink.org.cn/luwanzhou/gitlink-cli.git 2>/dev/null || true
git remote add muel https://www.gitlink.org.cn/muel/gitlink-cli.git 2>/dev/null || true
git remote add ohanabi https://www.gitlink.org.cn/ohanabi/gitlink-cli.git 2>/dev/null || true
git remote add wangyue111 https://www.gitlink.org.cn/wangyue111/gitlink-cli.git 2>/dev/null || true
git remote add wdgde https://www.gitlink.org.cn/wdgde/gitlink-cli.git 2>/dev/null || true
git remote add ylly https://www.gitlink.org.cn/ylly/gitlink-cli.git 2>/dev/null || true
git remote add yly5 https://www.gitlink.org.cn/yly5/gitlink-cli.git 2>/dev/null || true

echo "Fetching all forks..."
git fetch lijiabin1234 2>/dev/null || echo "  warn: Lijiabin1234 fetch failed"
git fetch maxwell7822 2>/dev/null || echo "  warn: Maxwell7822 fetch failed"
git fetch mengz 2>/dev/null || echo "  warn: Mengz fetch failed"
git fetch surponess 2>/dev/null || echo "  warn: Surponess fetch failed"
git fetch taoyouce 2>/dev/null || echo "  warn: Taoyouce fetch failed"
git fetch chroe 2>/dev/null || echo "  warn: chroe fetch failed"
git fetch co63oc 2>/dev/null || echo "  warn: co63oc fetch failed"
git fetch dtwdtw 2>/dev/null || echo "  warn: dtwdtw fetch failed"
git fetch heke1228 2>/dev/null || echo "  warn: heke1228 fetch failed"
git fetch jiangtx 2>/dev/null || echo "  warn: jiangtx fetch failed"
git fetch luwanzhou 2>/dev/null || echo "  warn: luwanzhou fetch failed"
git fetch muel 2>/dev/null || echo "  warn: muel fetch failed"
git fetch ohanabi 2>/dev/null || echo "  warn: ohanabi fetch failed"
git fetch wangyue111 2>/dev/null || echo "  warn: wangyue111 fetch failed"
git fetch wdgde 2>/dev/null || echo "  warn: wdgde fetch failed"
git fetch ylly 2>/dev/null || echo "  warn: ylly fetch failed"
git fetch yly5 2>/dev/null || echo "  warn: yly5 fetch failed"

echo "开始合并 139 个 PR..."
MERGED=0
FAILED=0

# PR #420
echo "=== PR #420: fix(api): 单次调用支持 :owner/:repo 占位符替换（修复 issue #20） ==="
if git merge --no-ff taoyouce/fix/api-single-call-placeholders -m "Merge PR #420: fix(api): 单次调用支持 :owner/:repo 占位符替换（修复 issue #20）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #414
echo "=== PR #414: feat(workflow): add release notes command ==="
if git merge --no-ff maxwell7822/feat/workflow-release-notes-pr -m "Merge PR #414: feat(workflow): add release notes command" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #396
echo "=== PR #396: feat(shortcut): 新增 capability 命令组 + internal/capability（后端 A ==="
if git merge --no-ff jiangtx/pr-shortcut-capability -m "Merge PR #396: feat(shortcut): 新增 capability 命令组 + internal/capability（后端 A" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #391
echo "=== PR #391: feat(shortcut): 新增 notification 命令组（通知 list/read/read-all/wa ==="
if git merge --no-ff jiangtx/pr-shortcut-notification -m "Merge PR #391: feat(shortcut): 新增 notification 命令组（通知 list/read/read-all/wa" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #390
echo "=== PR #390: feat(shortcut): 新增 pm 命令组（项目管理：仪表盘/Sprint/周报等） ==="
if git merge --no-ff jiangtx/pr-shortcut-pm -m "Merge PR #390: feat(shortcut): 新增 pm 命令组（项目管理：仪表盘/Sprint/周报等）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #389
echo "=== PR #389: 新增 export 命令组（issues/prs/contributors 导出 CSV/JSON） ==="
if git merge --no-ff jiangtx/pr-shortcut-export -m "Merge PR #389: 新增 export 命令组（issues/prs/contributors 导出 CSV/JSON）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #373
echo "=== PR #373: feat(repo): add repo +edit to update repository settings ==="
if git merge --no-ff taoyouce/feat/repo-edit -m "Merge PR #373: feat(repo): add repo +edit to update repository settings" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #366
echo "=== PR #366: feat(api): add --paginate and unwrap GitLink list responses  ==="
if git merge --no-ff heke1228/feat/api-paginate -m "Merge PR #366: feat(api): add --paginate and unwrap GitLink list responses " 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #365
echo "=== PR #365: feat(pr): surface merged_at in pr +view ==="
if git merge --no-ff heke1228/feat/pr-view-merged-at -m "Merge PR #365: feat(pr): surface merged_at in pr +view" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #361
echo "=== PR #361: feat(pr): add pr +edit to update an open pull request ==="
if git merge --no-ff heke1228/feat/pr-edit -m "Merge PR #361: feat(pr): add pr +edit to update an open pull request" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #358
echo "=== PR #358: feat(browse): add browse command group to open repo pages in ==="
if git merge --no-ff heke1228/b2-browse -m "Merge PR #358: feat(browse): add browse command group to open repo pages in" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #328
echo "=== PR #328: feat(skills): add research assistant skill suite ==="
if git merge --no-ff ohanabi/feat/research-skills-suite -m "Merge PR #328: feat(skills): add research assistant skill suite" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #313
echo "=== PR #313: feat(release): 增加发布资产查看与下载能力 ==="
if git merge --no-ff mengz/mengz/release-download-shortcuts-pr -m "Merge PR #313: feat(release): 增加发布资产查看与下载能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #312
echo "=== PR #312: feat(pr): 增加行级审查评论快捷命令 ==="
if git merge --no-ff mengz/mengz/pr-inline-review-comment-shortcuts-pr -m "Merge PR #312: feat(pr): 增加行级审查评论快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #308
echo "=== PR #308: 新增项目邀请管理模块 ==="
if git merge --no-ff wdgde/feature/project-invite -m "Merge PR #308: 新增项目邀请管理模块" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #285
echo "=== PR #285: feat(pr): add --login flag to filter list by issue author ==="
if git merge --no-ff co63oc/fix13 -m "Merge PR #285: feat(pr): add --login flag to filter list by issue author" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #280
echo "=== PR #280: feat(skills): add research trust skill ==="
if git merge --no-ff lijiabin1234/feat/research-trust-skill -m "Merge PR #280: feat(skills): add research trust skill" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #277
echo "=== PR #277: 新增 GitLink PR 队列评估与执行验证 Skill ==="
if git merge --no-ff mengz/mengz/gitlink-pr-assessor-skill -m "Merge PR #277: 新增 GitLink PR 队列评估与执行验证 Skill" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #270
echo "=== PR #270: feat(message-settings): 增加消息通知设置快捷命令 ==="
if git merge --no-ff mengz/mengz/message-settings-shortcuts -m "Merge PR #270: feat(message-settings): 增加消息通知设置快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #244
echo "=== PR #244: feat(skills): 新增科研主体画像 Skill（gitlink-research-profile） ==="
if git merge --no-ff luwanzhou/feat/research-profile-skill -m "Merge PR #244: feat(skills): 新增科研主体画像 Skill（gitlink-research-profile）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #216
echo "=== PR #216: feat(api): support saved variables in batch plans ==="
if git merge --no-ff wangyue111/wangyue111:feat/api-batch-save-vars -m "Merge PR #216: feat(api): support saved variables in batch plans" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #210
echo "=== PR #210: feat(skills): 新增维护者交接与分支治理 Skills ==="
if git merge --no-ff mengz/mengz/maintainer-skills -m "Merge PR #210: feat(skills): 新增维护者交接与分支治理 Skills" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #209
echo "=== PR #209: feat(milestone): 增加里程碑进度分析快捷命令 ==="
if git merge --no-ff mengz/mengz/milestone-report-shortcut -m "Merge PR #209: feat(milestone): 增加里程碑进度分析快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #203
echo "=== PR #203: feat(user): 增加用户画像分析快捷命令 ==="
if git merge --no-ff mengz/mengz/user-analytics-shortcuts -m "Merge PR #203: feat(user): 增加用户画像分析快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #194
echo "=== PR #194: fix(pr): 补齐 pr +view 的合并与关闭时间字段 ==="
if git merge --no-ff mengz/mengz/pr-view-timestamps -m "Merge PR #194: fix(pr): 补齐 pr +view 的合并与关闭时间字段" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #164
echo "=== PR #164: Add pull request review comment management shortcuts ==="
if git merge --no-ff mengz/mengz/pr-conversation-management -m "Merge PR #164: Add pull request review comment management shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #123
echo "=== PR #123: 新增 Release 资产下载命令 ==="
if git merge --no-ff mengz/mengz/release-download-shortcuts -m "Merge PR #123: 新增 Release 资产下载命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #99
echo "=== PR #99: 新增 5 个仓库检查快捷命令 (languages/contributors/files/tags/commits) ==="
if git merge --no-ff jiangtx/pr/repo-shortcuts -m "Merge PR #99: 新增 5 个仓库检查快捷命令 (languages/contributors/files/tags/commits)" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #405
echo "=== PR #405: feat(shortcut): 新增 watch 命令模块，支持仓库关注 ==="
if git merge --no-ff chroe/feat/watch-shortcut -m "Merge PR #405: feat(shortcut): 新增 watch 命令模块，支持仓库关注" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #404
echo "=== PR #404: feat(shortcut): 新增 file 命令模块（文件/目录操作） ==="
if git merge --no-ff chroe/feat/file-shortcut -m "Merge PR #404: feat(shortcut): 新增 file 命令模块（文件/目录操作）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #392
echo "=== PR #392: feat(cmd): 新增 alias/browse/status 开发者体验命令 ==="
if git merge --no-ff jiangtx/pr-cmd-dx-commands -m "Merge PR #392: feat(cmd): 新增 alias/browse/status 开发者体验命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #354
echo "=== PR #354: feat(auth): 新增 auth token 子命令与 auth status --show-token（对标 g ==="
if git merge --no-ff taoyouce/feat/auth-token-ux -m "Merge PR #354: feat(auth): 新增 auth token 子命令与 auth status --show-token（对标 g" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #318
echo "=== PR #318:  feat(label): add guarded batch label shortcuts ==="
if git merge --no-ff ohanabi/feat/label-batch-safety -m "Merge PR #318:  feat(label): add guarded batch label shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #281
echo "=== PR #281: fix(api): 补齐单次请求模板变量与请求头支持 ==="
if git merge --no-ff mengz/mengz/api-single-call-templates -m "Merge PR #281: fix(api): 补齐单次请求模板变量与请求头支持" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #279
echo "=== PR #279: feat(workflow): 新增陈旧队列扫描报告命令 ==="
if git merge --no-ff mengz/mengz/workflow-stale -m "Merge PR #279: feat(workflow): 新增陈旧队列扫描报告命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #272
echo "=== PR #272: feat(message): 补齐通知中心与消息设置管理工作流 ==="
if git merge --no-ff mengz/mengz/message-center-shortcuts-v2 -m "Merge PR #272: feat(message): 补齐通知中心与消息设置管理工作流" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #268
echo "=== PR #268: feat(branch): add lifecycle shortcuts ==="
if git merge --no-ff wangyue111/feat/branch-lifecycle-shortcuts -m "Merge PR #268: feat(branch): add lifecycle shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #234
echo "=== PR #234: feat(api): 增强单次调用模板变量与预演能力 ==="
if git merge --no-ff mengz/mengz/api-command-templates -m "Merge PR #234: feat(api): 增强单次调用模板变量与预演能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #221
echo "=== PR #221: feat(repo): 补齐仓库文件读取、搜索与批量提交工作流 ==="
if git merge --no-ff mengz/mengz/repo-file-workflow-shortcuts -m "Merge PR #221: feat(repo): 补齐仓库文件读取、搜索与批量提交工作流" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #217
echo "=== PR #217: feat(commands): add command catalog export ==="
if git merge --no-ff wangyue111/wangyue111:feat/commands-catalog -m "Merge PR #217: feat(commands): add command catalog export" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #211
echo "=== PR #211: feat(repo): add profile view shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/repo-profile-shortcuts -m "Merge PR #211: feat(repo): add profile view shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #199
echo "=== PR #199: feat(code): add read-only code browsing shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/code-read-shortcuts -m "Merge PR #199: feat(code): add read-only code browsing shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #190
echo "=== PR #190: Add workflow release readiness gate ==="
if git merge --no-ff mengz/mengz/workflow-release-readiness -m "Merge PR #190: Add workflow release readiness gate" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #189
echo "=== PR #189: Add workflow duplicate issue detection ==="
if git merge --no-ff mengz/mengz/workflow-issue-dedupe -m "Merge PR #189: Add workflow duplicate issue detection" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #188
echo "=== PR #188: Add workflow dependency risk audit ==="
if git merge --no-ff mengz/mengz/workflow-dependency-audit -m "Merge PR #188: Add workflow dependency risk audit" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #185
echo "=== PR #185: Add workflow pull request review queue ==="
if git merge --no-ff mengz/mengz/workflow-review-queue -m "Merge PR #185: Add workflow pull request review queue" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #184
echo "=== PR #184: Add workflow release notes generator ==="
if git merge --no-ff mengz/mengz/workflow-release-notes -m "Merge PR #184: Add workflow release notes generator" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #163
echo "=== PR #163: Add complete issue comment management shortcuts ==="
if git merge --no-ff mengz/mengz/issue-comment-management -m "Merge PR #163: Add complete issue comment management shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #158
echo "=== PR #158: Add code trace analysis shortcuts ==="
if git merge --no-ff mengz/mengz/trace-shortcuts -m "Merge PR #158: Add code trace analysis shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #150
echo "=== PR #150: 新增 Issue 批量导出命令 ==="
if git merge --no-ff mengz/mengz/issue-export -m "Merge PR #150: 新增 Issue 批量导出命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #101
echo "=== PR #101: 查看用户项目动态 ==="
if git merge --no-ff jiangtx/pr/user-shortcuts -m "Merge PR #101: 查看用户项目动态" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #100
echo "=== PR #100: 查看指定时间范围的开发统计 ==="
if git merge --no-ff jiangtx/pr/pr-shortcuts -m "Merge PR #100: 查看指定时间范围的开发统计" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #86
echo "=== PR #86: fix: preserve issue metadata on update ==="
if git merge --no-ff dtwdtw/fix/issue-142625-preserve-issue-metadata -m "Merge PR #86: fix: preserve issue metadata on update" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #67
echo "=== PR #67: feat(repo): add repository units shortcuts ==="
if git merge --no-ff mengz/mengz/repo-units-shortcut -m "Merge PR #67: feat(repo): add repository units shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #60
echo "=== PR #60: feat: add notification shortcuts ==="
if git merge --no-ff mengz/mengz/notification-shortcut -m "Merge PR #60: feat: add notification shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #56
echo "=== PR #56: feat: add git tag shortcut group ==="
if git merge --no-ff mengz/mengz/tag-shortcut -m "Merge PR #56: feat: add git tag shortcut group" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #23
echo "=== PR #23: feat: support fork metadata in pr create ==="
if git merge --no-ff mengz/codex/pr-create-fork-support -m "Merge PR #23: feat: support fork metadata in pr create" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #395
echo "=== PR #395: feat(skills): add maintainer copilot skill ==="
if git merge --no-ff yly5/feat/maintainer-copilot-skill -m "Merge PR #395: feat(skills): add maintainer copilot skill" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #370
echo "=== PR #370: feat(pr): add pr +checks showing CI build status for a pull  ==="
if git merge --no-ff heke1228/feat/pr-checks -m "Merge PR #370: feat(pr): add pr +checks showing CI build status for a pull " 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #357
echo "=== PR #357: feat(skills): add SKILL.md metadata validator and CI gate. ==="
if git merge --no-ff heke1228/b1-skills-governance -m "Merge PR #357: feat(skills): add SKILL.md metadata validator and CI gate." 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #319
echo "=== PR #319: feat(pr): add merge readiness and branch helper shortcuts ==="
if git merge --no-ff ohanabi/feat/pr-readiness-shortcuts -m "Merge PR #319: feat(pr): add merge readiness and branch helper shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #310
echo "=== PR #310: 贡献者新增图表显示 ==="
if git merge --no-ff wdgde/feature/code-stats -m "Merge PR #310: 贡献者新增图表显示" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #303
echo "=== PR #303: gitlink-cli 新增 11 个仓库设置相关快捷命令 ==="
if git merge --no-ff wdgde/feat/repo-settings-shortcuts -m "Merge PR #303: gitlink-cli 新增 11 个仓库设置相关快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #300
echo "=== PR #300: 新增批量重开、标签、指派、评论、导出、导入命令 ==="
if git merge --no-ff wdgde/feature/issue-batch-enhance -m "Merge PR #300: 新增批量重开、标签、指派、评论、导出、导入命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #273
echo "=== PR #273: feat(repo): 补齐仓库治理中的转移与模块配置能力 ==="
if git merge --no-ff mengz/mengz/repo-governance-shortcuts -m "Merge PR #273: feat(repo): 补齐仓库治理中的转移与模块配置能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #227
echo "=== PR #227: feat(message): add inbox management shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/message-inbox-shortcuts -m "Merge PR #227: feat(message): add inbox management shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #225
echo "=== PR #225: feat(repo): add scaffold creation options ==="
if git merge --no-ff wangyue111/wangyue111:feat/repo-scaffold-options -m "Merge PR #225: feat(repo): add scaffold creation options" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #224
echo "=== PR #224: feat(project-template): add issue template shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/project-template-lifecycle -m "Merge PR #224: feat(project-template): add issue template shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #223
echo "=== PR #223: feat(public-key): add SSH key shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/account-public-key-shortcuts -m "Merge PR #223: feat(public-key): add SSH key shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #222
echo "=== PR #222: feat(attachment): 新增附件上传与删除快捷命令 ==="
if git merge --no-ff mengz/mengz/attachment-shortcut -m "Merge PR #222: feat(attachment): 新增附件上传与删除快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #214
echo "=== PR #214: feat(pr): add conversation comment shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/pr-conversation-shortcuts -m "Merge PR #214: feat(pr): add conversation comment shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #200
echo "=== PR #200: feat(pr): add review journal shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/pr-journal-shortcuts -m "Merge PR #200: feat(pr): add review journal shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #198
echo "=== PR #198: feat(message): 增加消息中心快捷命令 ==="
if git merge --no-ff mengz/mengz/message-center-shortcuts -m "Merge PR #198: feat(message): 增加消息中心快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #196
echo "=== PR #196: feat(release): 增加发布资产管理快捷命令 ==="
if git merge --no-ff mengz/mengz/release-asset-shortcuts -m "Merge PR #196: feat(release): 增加发布资产管理快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #186
echo "=== PR #186: feat(ref): add branch and tag shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/ref-shortcuts -m "Merge PR #186: feat(ref): add branch and tag shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #160
echo "=== PR #160: Add GitLink feedback submission shortcut ==="
if git merge --no-ff mengz/mengz/feedback-shortcut -m "Merge PR #160: Add GitLink feedback submission shortcut" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #78
echo "=== PR #78: feat(branch): complete OpenAPI shortcuts ==="
if git merge --no-ff wangyue111/feat/branch-openapi-shortcuts -m "Merge PR #78: feat(branch): complete OpenAPI shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #73
echo "=== PR #73: feat(user): add SSH key shortcuts ==="
if git merge --no-ff mengz/mengz/user-key-shortcuts -m "Merge PR #73: feat(user): add SSH key shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #403
echo "=== PR #403: feat(shortcut): 新增 commit 命令模块（提交历史/diff/blame） ==="
if git merge --no-ff chroe/feat/commit-shortcut -m "Merge PR #403: feat(shortcut): 新增 commit 命令模块（提交历史/diff/blame）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #383
echo "=== PR #383: feat(pr): add +commits and +check-merge ==="
if git merge --no-ff taoyouce/feat/pr-commits-checkmerge -m "Merge PR #383: feat(pr): add +commits and +check-merge" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #380
echo "=== PR #380: feat(action): new command group for Gitea Actions workflows ==="
if git merge --no-ff taoyouce/feat/actions-group -m "Merge PR #380: feat(action): new command group for Gitea Actions workflows" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #330
echo "=== PR #330: feat(file): 新增 file 快捷命令组，无需克隆即可读写仓库文件 ==="
if git merge --no-ff taoyouce/feat/file-shortcuts -m "Merge PR #330: feat(file): 新增 file 快捷命令组，无需克隆即可读写仓库文件" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #321
echo "=== PR #321: feat(issue): add journal and activity shortcuts ==="
if git merge --no-ff ohanabi/feat/issue-journal-shortcuts -m "Merge PR #321: feat(issue): add journal and activity shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #315
echo "=== PR #315: feat(notification): add message notification shortcuts ==="
if git merge --no-ff ohanabi/feat/notification-shortcuts -m "Merge PR #315: feat(notification): add message notification shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #309
echo "=== PR #309: 新增 health diagnose 命令，实现项目健康度综合诊断功能 ==="
if git merge --no-ff wdgde/feature/health-check -m "Merge PR #309: 新增 health diagnose 命令，实现项目健康度综合诊断功能" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #271
echo "=== PR #271: feat(issue): 补齐 Issue 批量导出与评论管理工作流 ==="
if git merge --no-ff mengz/mengz/issue-workflow-shortcuts -m "Merge PR #271: feat(issue): 补齐 Issue 批量导出与评论管理工作流" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #263
echo "=== PR #263: 新增 template 命令，支持 GitLink 平台项目模板的增删改查操作 ==="
if git merge --no-ff wdgde/feature/template -m "Merge PR #263: 新增 template 命令，支持 GitLink 平台项目模板的增删改查操作" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #239
echo "=== PR #239: feat(transfer): add incoming transfer shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/transfer-inbox-shortcuts -m "Merge PR #239: feat(transfer): add incoming transfer shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #226
echo "=== PR #226: feat(issue): 增加批量评论、批量更新与文件正文输入能力 ==="
if git merge --no-ff mengz/mengz/issue-batch-ops -m "Merge PR #226: feat(issue): 增加批量评论、批量更新与文件正文输入能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #208
echo "=== PR #208: feat(user): add pinned project shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/user-pin-shortcuts -m "Merge PR #208: feat(user): add pinned project shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #204
echo "=== PR #204: feat(org): 增强组织团队与成员管理快捷命令 ==="
if git merge --no-ff mengz/mengz/org-team-shortcuts -m "Merge PR #204: feat(org): 增强组织团队与成员管理快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #181
echo "=== PR #181: feat(topic): add project topic shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/project-topic-shortcuts -m "Merge PR #181: feat(topic): add project topic shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #180
echo "=== PR #180: feat(template): add project template shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/project-template-shortcuts -m "Merge PR #180: feat(template): add project template shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #147
echo "=== PR #147: feat(shortcuts): 新增 wiki/commit/file/star/watch 模块及批量操作 ==="
if git merge --no-ff chroe/pr/shortcuts-modules -m "Merge PR #147: feat(shortcuts): 新增 wiki/commit/file/star/watch 模块及批量操作" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #113
echo "=== PR #113: feat(todo): add request approval shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/todo-openapi-shortcuts -m "Merge PR #113: feat(todo): add request approval shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #77
echo "=== PR #77: feat(journal): add issue and PR comment shortcuts ==="
if git merge --no-ff wangyue111/feat/journal-openapi-shortcuts -m "Merge PR #77: feat(journal): add issue and PR comment shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #374
echo "=== PR #374: feat(issue): add +comments, +comment-edit, +comment-delete f ==="
if git merge --no-ff taoyouce/feat/issue-comment-management -m "Merge PR #374: feat(issue): add +comments, +comment-edit, +comment-delete f" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #320
echo "=== PR #320: feat(repo): add raw file content shortcuts ==="
if git merge --no-ff ohanabi/feat/repo-raw-file-shortcuts -m "Merge PR #320: feat(repo): add raw file content shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #316
echo "=== PR #316: feat(pm): add project management shortcuts ==="
if git merge --no-ff ohanabi/feat/pm-shortcuts -m "Merge PR #316: feat(pm): add project management shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #282
echo "=== PR #282: feat(branch): 补齐分支筛选、默认分支切换与恢复能力 ==="
if git merge --no-ff mengz/mengz/branch-lifecycle-shortcuts -m "Merge PR #282: feat(branch): 补齐分支筛选、默认分支切换与恢复能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #274
echo "=== PR #274: feat(shortcuts): add template shortcuts ==="
if git merge --no-ff co63oc/fix11 -m "Merge PR #274: feat(shortcuts): add template shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #212
echo "=== PR #212: feat(feedback): add feedback shortcut ==="
if git merge --no-ff wangyue111/wangyue111:feat/feedback-shortcut -m "Merge PR #212: feat(feedback): add feedback shortcut" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #201
echo "=== PR #201: feat(account): add account auth shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/account-auth-shortcuts -m "Merge PR #201: feat(account): add account auth shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #192
echo "=== PR #192: feat(repo): add navigation unit shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/repo-units-shortcuts -m "Merge PR #192: feat(repo): add navigation unit shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #187
echo "=== PR #187: feat(org): add team project bulk shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/org-team-project-shortcuts -m "Merge PR #187: feat(org): add team project bulk shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #182
echo "=== PR #182: feat(issue): add journal maintenance shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/issue-journal-shortcuts -m "Merge PR #182: feat(issue): add journal maintenance shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #178
echo "=== PR #178: feat(contents): add repository content shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/contents-shortcuts -m "Merge PR #178: feat(contents): add repository content shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #176
echo "=== PR #176: feat(user): add dashboard shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/user-dashboard-shortcuts -m "Merge PR #176: feat(user): add dashboard shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #175
echo "=== PR #175: feat(notification): add message and setting shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/notification-shortcuts -m "Merge PR #175: feat(notification): add message and setting shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #173
echo "=== PR #173: feat(account): add cancellation shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/account-cancel-shortcuts -m "Merge PR #173: feat(account): add cancellation shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #171
echo "=== PR #171: feat(oauth): add token shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/oauth-token-shortcuts -m "Merge PR #171: feat(oauth): add token shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #167
echo "=== PR #167: feat(account): add email verification shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/account-email-shortcuts -m "Merge PR #167: feat(account): add email verification shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #142
echo "=== PR #142: 新增 PR 本地检出命令 ==="
if git merge --no-ff mengz/mengz/pr-checkout -m "Merge PR #142: 新增 PR 本地检出命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #135
echo "=== PR #135: feat(dev): add developer resource shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/developer-resource-shortcuts -m "Merge PR #135: feat(dev): add developer resource shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #118
echo "=== PR #118: feat(access): add project access shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/access-openapi-shortcuts -m "Merge PR #118: feat(access): add project access shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #114
echo "=== PR #114: feat(mirror): add mirror repository shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/mirror-openapi-shortcuts -m "Merge PR #114: feat(mirror): add mirror repository shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #82
echo "=== PR #82: feat(meta): add attachment and metadata shortcuts ==="
if git merge --no-ff wangyue111/feat/meta-attachment-shortcuts -m "Merge PR #82: feat(meta): add attachment and metadata shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #76
echo "=== PR #76: feat(notification): add OpenAPI shortcuts ==="
if git merge --no-ff wangyue111/feat/notification-openapi-shortcuts -m "Merge PR #76: feat(notification): add OpenAPI shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #387
echo "=== PR #387: feat(commit): add commit group (+list, +recent, +diff, +file ==="
if git merge --no-ff taoyouce/feat/commit-group -m "Merge PR #387: feat(commit): add commit group (+list, +recent, +diff, +file" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #323
echo "=== PR #323: feat(user): add user statistics shortcuts ==="
if git merge --no-ff ohanabi/feat/user-statistics-shortcuts -m "Merge PR #323: feat(user): add user statistics shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #314
echo "=== PR #314: feat(repo,pr): add code history and batch file shortcuts ==="
if git merge --no-ff ohanabi/feat/code-history-file-shortcuts -m "Merge PR #314: feat(repo,pr): add code history and batch file shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #261
echo "=== PR #261: 添加 gitlink-cli auth checkin 命令，用于定时刷新认证会话，防止用户登录状态过期 ==="
if git merge --no-ff wdgde/feature/wdd -m "Merge PR #261: 添加 gitlink-cli auth checkin 命令，用于定时刷新认证会话，防止用户登录状态过期" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #70
echo "=== PR #70: feat(user): add account and stats shortcuts ==="
if git merge --no-ff wangyue111/feat/user-account-stats-shortcuts -m "Merge PR #70: feat(user): add account and stats shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #353
echo "=== PR #353: feat(attachment): 新增 attachment +upload/+download 附件上传下载命令组 ==="
if git merge --no-ff taoyouce/feat/attachment-upload -m "Merge PR #353: feat(attachment): 新增 attachment +upload/+download 附件上传下载命令组" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #347
echo "=== PR #347: feat(i18n): 10 个命令组接入 i18n 全覆盖（145 个中英键） ==="
if git merge --no-ff taoyouce/feat/i18n-coverage -m "Merge PR #347: feat(i18n): 10 个命令组接入 i18n 全覆盖（145 个中英键）" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #159
echo "=== PR #159: Add member application workflow shortcuts ==="
if git merge --no-ff mengz/mengz/project-application-requests -m "Merge PR #159: Add member application workflow shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #115
echo "=== PR #115: feat: add catalog template shortcuts ==="
if git merge --no-ff mengz/mengz/catalog-shortcuts -m "Merge PR #115: feat: add catalog template shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #65
echo "=== PR #65: feat(wiki): add OpenAPI shortcuts ==="
if git merge --no-ff wangyue111/feat/wiki-openapi-shortcuts -m "Merge PR #65: feat(wiki): add OpenAPI shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #58
echo "=== PR #58: feat: add repository reaction shortcuts ==="
if git merge --no-ff mengz/mengz/reaction-shortcut -m "Merge PR #58: feat: add repository reaction shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #242
echo "=== PR #242: feat: 新增 wiki 管理快捷命令 ==="
if git merge --no-ff luwanzhou/feat/wiki-shortcuts -m "Merge PR #242: feat: 新增 wiki 管理快捷命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #179
echo "=== PR #179: feat(dataset): add research dataset shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/dataset-shortcuts -m "Merge PR #179: feat(dataset): add research dataset shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #177
echo "=== PR #177: feat(wiki): add wiki management shortcuts ==="
if git merge --no-ff wangyue111/wangyue111:feat/wiki-shortcuts -m "Merge PR #177: feat(wiki): add wiki management shortcuts" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #408
echo "=== PR #408: 提交pr ==="
if git merge --no-ff surponess/master -m "Merge PR #408: 提交pr" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #349
echo "=== PR #349: feat: 新增Wiki/Label/Notification命令 + 批量处理命令+Snippet/自然语言do创新  ==="
if git merge --no-ff ylly/master -m "Merge PR #349: feat: 新增Wiki/Label/Notification命令 + 批量处理命令+Snippet/自然语言do创新 " 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #346
echo "=== PR #346: # feat(feishu): 新增分层飞书协作导出能力 ==="
if git merge --no-ff muel/feat/feishu-export-clean -m "Merge PR #346: # feat(feishu): 新增分层飞书协作导出能力" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #338
echo "=== PR #338: feat(list): issue/pr/branch/release +list 新增 --all 自动翻页（翻页助手 ==="
if git merge --no-ff taoyouce/feat/list-all-pagination -m "Merge PR #338: feat(list): issue/pr/branch/release +list 新增 --all 自动翻页（翻页助手" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #305
echo "=== PR #305: test: 代码质量看门人演示 PR ==="
if git merge --no-ff ylly/test/code-review-demo -m "Merge PR #305: test: 代码质量看门人演示 PR" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #103
echo "=== PR #103: feat: 新建 pm 模块，添加 6 条项目管理命令 ==="
if git merge --no-ff jiangtx/wyx_branch -m "Merge PR #103: feat: 新建 pm 模块，添加 6 条项目管理命令" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

# PR #50
echo "=== PR #50: feat: add wiki shortcut group ==="
if git merge --no-ff mengz/mengz/wiki-shortcut -m "Merge PR #50: feat: add wiki shortcut group" 2>/dev/null; then
  echo "  ✅ clean merge"
  MERGED=1
else
  # 解冲突: accept theirs for all conflicted files
  CONFLICTED=
  if [ -z "" ]; then
    echo "  ❌ merge failed (not conflict)"
    git merge --abort 2>/dev/null || true
    FAILED=1
  else
    for f in ; do
      git checkout --theirs "" 2>/dev/null && git add ""
    done
    if git commit --no-edit 2>/dev/null; then
      echo "  ✅ conflict resolved"
      MERGED=1
    else
      git merge --abort 2>/dev/null || true
      echo "  ❌ commit failed"
      FAILED=1
    fi
  fi
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "完成: 合并 , 失败 "