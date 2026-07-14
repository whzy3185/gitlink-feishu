import json, os, csv
from collections import defaultdict

REPOS = [
    "NSCCN/AlphaFold3",
    "Supercomputing/alphafold",
    "NSCCN/AlphaFill",
    "NSCCN/RoseTTAFold",
    "scnc/openfold"
]

DATA_DIR = "data/raw"
OUT_DIR = "data/processed"
os.makedirs(OUT_DIR, exist_ok=True)

def load_json(repo, suffix):
    safe = repo.replace("/", "_")
    path = os.path.join(DATA_DIR, f"{safe}_{suffix}.json")
    with open(path, encoding="utf-8") as f:
        return json.load(f)

# 1. 仓库基本信息
with open(os.path.join(OUT_DIR, "repo_summary.csv"), "w", newline="", encoding="utf-8") as f:
    writer = csv.writer(f)
    writer.writerow(["repo", "full_name", "size", "watchers", "forks", "mirror", "default_branch", "description"])
    for repo in REPOS:
        info = load_json(repo, "info")["data"]
        writer.writerow([repo, info.get("full_name"), info.get("size"), info.get("watchers_count", 0),
                         info.get("forked_count", 0), info.get("mirror"), info.get("default_branch", ""),
                         info.get("description", "")])

# 2. 代码统计与贡献者 Top3
with open(os.path.join(OUT_DIR, "code_stats.csv"), "w", newline="", encoding="utf-8") as f:
    writer = csv.writer(f)
    writer.writerow(["repo", "total_additions", "total_deletions", "commit_count", "author_count",
                     "top1_author", "top1_commits", "top2_author", "top2_commits", "top3_author", "top3_commits"])
    for repo in REPOS:
        stats = load_json(repo, "code_stats")["data"]
        authors = stats.get("authors", [])
        sorted_authors = sorted(authors, key=lambda x: x["commits"], reverse=True)
        top3 = sorted_authors[:3]
        row = [repo, stats.get("additions", 0), stats.get("deletions", 0),
               stats.get("commit_count", 0), stats.get("author_count", 0)]
        for i in range(3):
            if i < len(top3):
                row.append(top3[i]["author"]["name"])
                row.append(top3[i]["commits"])
            else:
                row.extend(["", ""])
        writer.writerow(row)

# 3. 贡献者网络
contrib_repo_map = defaultdict(set)
for repo in REPOS:
    conts = load_json(repo, "contributors")["data"]["list"]
    for c in conts:
        email = c.get("email", "").strip().lower()
        if email:
            contrib_repo_map[email].add(repo)

with open(os.path.join(OUT_DIR, "contributor_network.csv"), "w", newline="", encoding="utf-8") as f:
    writer = csv.writer(f)
    writer.writerow(["email", "repo_count", "repo_list"])
    for email, repos_set in contrib_repo_map.items():
        writer.writerow([email, len(repos_set), "; ".join(sorted(repos_set))])

# 4. 语言分布
all_langs = set()
repo_lang_data = {}
for repo in REPOS:
    langs = load_json(repo, "languages")["data"]
    repo_lang_data[repo] = langs
    all_langs.update(langs.keys())

with open(os.path.join(OUT_DIR, "language_distribution.csv"), "w", newline="", encoding="utf-8") as f:
    writer = csv.writer(f)
    writer.writerow(["repo"] + sorted(all_langs))
    for repo in REPOS:
        row = [repo]
        for lang in sorted(all_langs):
            row.append(repo_lang_data[repo].get(lang, "0%"))
        writer.writerow(row)

print("分析完成！CSV 已保存到 data/processed/")