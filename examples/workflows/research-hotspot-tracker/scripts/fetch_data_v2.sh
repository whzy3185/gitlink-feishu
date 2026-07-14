#!/bin/bash
mkdir -p data/raw

while IFS= read -r repo; do
    [ -z "$repo" ] && continue
    owner=$(echo $repo | cut -d/ -f1)
    name=$(echo $repo | cut -d/ -f2)
    safe=$(echo $repo | tr '/' '_')

    echo "正在采集: $repo"

    # 仓库基本信息
    gitlink-cli repo +info --owner "$owner" --repo "$name" --format json > "data/raw/${safe}_info.json"

    # 代码统计（含贡献者代码行数）
    gitlink-cli repo +code-stats --owner "$owner" --repo "$name" --format json > "data/raw/${safe}_code_stats.json"

    # 贡献者列表（含贡献百分比）
    gitlink-cli repo +contributors --owner "$owner" --repo "$name" --format json > "data/raw/${safe}_contributors.json"

    # 语言统计
    gitlink-cli repo +languages --owner "$owner" --repo "$name" --format json > "data/raw/${safe}_languages.json"

done < repos.txt

echo "采集完成！"