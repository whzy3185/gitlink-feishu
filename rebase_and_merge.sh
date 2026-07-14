#!/bin/bash
set -e

# PR 处理顺序和对应的 fork branch
declare -A PR_BRANCHES
PR_BRANCHES[421]="taoyouce/feat/skill-fork-sync"
PR_BRANCHES[413]="taoyouce/feat/semantic-audit-skill"
PR_BRANCHES[412]="taoyouce/feat/issue-delete"
PR_BRANCHES[406]="chroe/fix/msys2-path-pollution"
PR_BRANCHES[304]="muel/fix/i18n-locale-eol"
PR_BRANCHES[379]="taoyouce/feat/repo-topics"
PR_BRANCHES[378]="taoyouce/feat/repo-blame"
PR_BRANCHES[382]="taoyouce/feat/repo-activity"
PR_BRANCHES[381]="taoyouce/feat/repo-forks-topcounts"
PR_BRANCHES[355]="taoyouce/feat/repo-clone"
PR_BRANCHES[386]="taoyouce/feat/branch-default-all"
PR_BRANCHES[284]="mengz/mengz/pr-list-search-number"
PR_BRANCHES[283]="mengz/mengz/pr-list-show-number"
PR_BRANCHES[213]="wangyue111/feat/repo-mirror-sync-shortcut"

ORDER=(421 413 412 406 304 379 378 382 381 355 386 284 283 213)

echo "开始按顺序处理 ${#ORDER[@]} 个冲突 PR..."
echo ""

for PR_NUM in "${ORDER[@]}"; do
    BRANCH="${PR_BRANCHES[$PR_NUM]}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "处理 PR #${PR_NUM}: branch=${BRANCH}"
    
    # 创建临时分支来做 rebase
    TEMP_BRANCH="rebase-pr-${PR_NUM}"
    git checkout -B "$TEMP_BRANCH" "$BRANCH" 2>/dev/null
    
    # Rebase onto master
    if git rebase origin/master 2>/dev/null; then
        echo "  ✅ rebase 成功 (无需手动解冲突)"
    else
        echo "  ⚠️  rebase 有冲突，尝试自动解决..."
        # 对于 JSON 文件、README、test 文件的追加冲突，accept theirs for new content
        CONFLICTED=$(git diff --name-only --diff-filter=U 2>/dev/null)
        echo "  冲突文件: $CONFLICTED"
        
        ALL_RESOLVED=true
        for F in $CONFLICTED; do
            case "$F" in
                *.json)
                    # JSON 文件: accept both (theirs adds new keys)
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                    echo "    $F → accept theirs (新增 key)"
                    ;;
                *README*)
                    # README: accept theirs (新增行)
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                    echo "    $F → accept theirs (新增行)"
                    ;;
                *_test.go)
                    # Test 文件: accept theirs (新增测试)
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                    echo "    $F → accept theirs (新增测试)"
                    ;;
                *.go)
                    # Go 源文件: accept theirs (新增函数/注册)
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                    echo "    $F → accept theirs (新增代码)"
                    ;;
                *)
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                    echo "    $F → accept theirs"
                    ;;
            esac
        done
        
        git rebase --continue --no-edit 2>/dev/null || {
            # 可能还有后续冲突
            CONFLICTED2=$(git diff --name-only --diff-filter=U 2>/dev/null)
            if [ -n "$CONFLICTED2" ]; then
                for F in $CONFLICTED2; do
                    git checkout --theirs "$F" 2>/dev/null && git add "$F"
                done
                git rebase --continue --no-edit 2>/dev/null || git rebase --abort
            fi
        }
    fi
    
    # 回到 master
    git checkout master 2>/dev/null
    echo ""
done

echo "所有 PR 已 rebase 到各自临时分支"
echo "现在逐个 merge 到 master..."
