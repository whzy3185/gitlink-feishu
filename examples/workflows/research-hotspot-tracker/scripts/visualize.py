import pandas as pd
import matplotlib.pyplot as plt
import matplotlib
import os
from collections import Counter

matplotlib.rcParams['font.sans-serif'] = ['DejaVu Sans']  # 或系统支持的中文字体
matplotlib.rcParams['axes.unicode_minus'] = False

OUT_DIR = "output"
os.makedirs(OUT_DIR, exist_ok=True)

# 读取数据
stats = pd.read_csv("data/processed/code_stats.csv")
langs = pd.read_csv("data/processed/language_distribution.csv", index_col=0)
contrib = pd.read_csv("data/processed/contributor_network.csv")

# ========== 1. 代码量对比柱状图 ==========
fig, ax = plt.subplots(figsize=(12, 6))
x = range(len(stats))
width = 0.35
ax.bar(x, stats['total_additions']/1000, width, label='Additions (k lines)', color='steelblue')
ax.bar([i+width for i in x], stats['total_deletions']/1000, width, label='Deletions (k lines)', color='lightcoral')
ax.set_xticks([i+width/2 for i in x])
ax.set_xticklabels(stats['repo'], rotation=15, ha='right')
ax.set_ylabel('Lines (thousands)')
ax.set_title('Code Additions vs Deletions per Repository')
ax.legend()
plt.tight_layout()
plt.savefig(os.path.join(OUT_DIR, "code_volume_comparison.png"), dpi=150)
plt.close()

# ========== 2. 提交次数与贡献者人数气泡图 ==========
fig, ax = plt.subplots(figsize=(10, 6))
ax.scatter(stats['author_count'], stats['commit_count'], s=stats['total_additions']/500, alpha=0.6, c='darkgreen')
for i, row in stats.iterrows():
    ax.annotate(row['repo'], (row['author_count'], row['commit_count']), fontsize=8)
ax.set_xlabel('Number of Authors')
ax.set_ylabel('Commit Count')
ax.set_title('Repository Activity: Commits vs Authors (bubble size = additions)')
plt.tight_layout()
plt.savefig(os.path.join(OUT_DIR, "activity_bubble.png"), dpi=150)
plt.close()

# ========== 3. 语言分布堆叠条形图 ==========
# 将百分比字符串转为数值
lang_perc = langs.apply(lambda col: col.str.rstrip('%').astype(float))
ax = lang_perc.plot(kind='bar', stacked=True, figsize=(12, 6), colormap='Set3')
ax.set_ylabel('Percentage')
ax.set_title('Language Distribution Across Repositories')
ax.legend(loc='upper right', bbox_to_anchor=(1.15, 1))
plt.tight_layout()
plt.savefig(os.path.join(OUT_DIR, "language_distribution.png"), dpi=150)
plt.close()

# ========== 4. 贡献者网络图（前20位跨仓库贡献者） ==========
# 简单条形图展示贡献仓库数最多的贡献者
top_contrib = contrib[contrib['repo_count'] > 1].sort_values('repo_count', ascending=False).head(15)
if not top_contrib.empty:
    fig, ax = plt.subplots(figsize=(12, 6))
    ax.barh(top_contrib['email'], top_contrib['repo_count'], color='mediumpurple')
    ax.set_xlabel('Number of Repositories')
    ax.set_title('Top Contributors Across Multiple Repositories')
    plt.tight_layout()
    plt.savefig(os.path.join(OUT_DIR, "cross_repo_contributors.png"), dpi=150)
    plt.close()
else:
    print("No cross-repo contributors found.")

print("可视化图表已保存到 output/")