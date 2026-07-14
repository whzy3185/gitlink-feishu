#!/usr/bin/env python3
"""
本地批量解冲突合入：对每个 PR 执行 git merge --no-ff，遇到冲突时 accept theirs。
支持 add/add 冲突（两个分支都创建同名文件）。
"""
import json
import subprocess
import sys
import os

REPO_DIR = "/tmp/gitlink-cli-rebase"
os.chdir(REPO_DIR)

def run(cmd, check=True):
    r = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if check and r.returncode != 0:
        return None
    return r

def git(cmd, check=False):
    return run(f"git {cmd}", check=check)

def get_conflicted_files():
    """获取所有冲突文件（包括 add/add）"""
    r = git("status --porcelain")
    if not r:
        return []
    files = []
    for line in r.stdout.strip().split('\n'):
        if not line:
            continue
        status = line[:2]
        path = line[3:]
        # UU=both modified, AA=both added, DU/UD=delete conflicts
        if status in ('UU', 'AA', 'DU', 'UD', 'AU', 'UA'):
            files.append(path)
    return files

def resolve_conflicts():
    """Accept theirs for all conflicted files"""
    files = get_conflicted_files()
    if not files:
        return False

    for f in files:
        # For add/add conflicts, checkout --theirs may fail, use a different approach
        r = git(f'checkout --theirs "{f}"')
        if r is None or r.returncode != 0:
            # Fallback: just use what's in the index from theirs
            git(f'show :3:"{f}" > "{f}"', check=False)
        git(f'add "{f}"')
    return True

def merge_pr(remote, branch, number, title):
    """Merge one PR branch into current HEAD"""
    msg = f"Merge PR #{number}: {title[:60]}"

    # Try clean merge first
    r = git(f'merge --no-ff {remote}/{branch} -m "{msg}"')
    if r and r.returncode == 0:
        return "clean"

    # Check if we have conflicts
    files = get_conflicted_files()
    if not files:
        # Not a conflict, some other error (e.g., branch not found)
        git("merge --abort")
        return "error"

    # Resolve conflicts
    resolve_conflicts()

    # Commit the merge
    r = git(f'commit --no-edit')
    if r and r.returncode == 0:
        return "resolved"

    # If commit fails, abort
    git("merge --abort")
    return "failed"

def main():
    # Load PR list
    with open("/Users/baai/codebase/gitlink-cli/scripts/pr-triage/conflict_prs.json") as f:
        data = json.load(f)

    high_risk_files = ['internal/auth/', 'token_store', 'internal/client/client.go', 'cmd/auth/']
    safe = []
    for pr in data['conflict']:
        cfs = pr['conflict_files']
        is_risky = any(any(h in cf for h in high_risk_files) for cf in cfs)
        if not is_risky:
            safe.append(pr)

    safe.sort(key=lambda p: len(p['conflict_files']))

    # Collect unique fork remotes
    forks = set()
    for pr in safe:
        if pr['fork_login']:
            forks.add(pr['fork_login'])

    print(f"添加 {len(forks)} 个 fork remote 并 fetch...")
    for f in sorted(forks):
        git(f'remote add {f.lower()} https://www.gitlink.org.cn/{f}/gitlink-cli.git')
        r = git(f'fetch {f.lower()}')
        if r is None or r.returncode != 0:
            print(f"  ⚠️ fetch {f} 失败")

    print(f"\n开始合并 {len(safe)} 个 PR...")
    print("=" * 60)

    merged = 0
    failed = 0
    failed_prs = []

    for i, pr in enumerate(safe):
        num = pr['number']
        remote = pr['fork_login'].lower() if pr['fork_login'] else 'origin'
        branch = pr['head']
        title = pr['title']

        # Check branch exists
        r = git(f'rev-parse --verify {remote}/{branch}')
        if r is None or r.returncode != 0:
            print(f"  [{i+1}/{len(safe)}] #{num}: ❌ branch {remote}/{branch} not found")
            failed += 1
            failed_prs.append((num, "branch not found"))
            continue

        result = merge_pr(remote, branch, num, title)

        if result in ("clean", "resolved"):
            merged += 1
            tag = "✅" if result == "clean" else "✅ (conflict resolved)"
            print(f"  [{i+1}/{len(safe)}] #{num}: {tag}")
        else:
            failed += 1
            failed_prs.append((num, result))
            print(f"  [{i+1}/{len(safe)}] #{num}: ❌ {result}")

        # Progress summary every 20
        if (i + 1) % 20 == 0:
            print(f"    --- 进度: {merged} merged, {failed} failed ---")

    print("\n" + "=" * 60)
    print(f"完成: 合并 {merged}, 失败 {failed}")

    if failed_prs:
        print(f"\n失败的 PR:")
        for num, reason in failed_prs:
            print(f"  #{num}: {reason}")

    # Verify build
    print("\n验证编译...")
    r = run("go build ./...")
    if r and r.returncode == 0:
        print("  ✅ go build 通过")
    else:
        print("  ❌ go build 失败!")
        if r:
            print(r.stderr[:500])

if __name__ == "__main__":
    main()
