#!/bin/bash
echo "===== 科研仓库分析一键运行 ====="
echo "[1/3] 数据采集..."
bash scripts/fetch_data_v2.sh
echo "[2/3] 数据分析..."
python scripts/analyze_v2.py
echo "[3/3] 生成可视化图表..."
python scripts/visualize.py
echo "===== 完成！图表在 output/，数据在 data/processed/ ====="