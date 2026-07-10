# 科研场景应用报告

## 1. 技术实现

### 架构图
![架构图](autodl-tmp/gitlink-research-hotspot-tracker/output/struct.svg)

### 工具链
- gitlink-cli：获取仓库元数据、代码统计、贡献者、语言分布
- Python：pandas 数据处理 + matplotlib 可视化
- AI Agent：智谱清言 / 百度 Comate（Skill 验证平台）

## 2. 科研赋能价值
1. 项目评估：通过代码量、提交频率、贡献者集中度快速判断开源项目健康度
2. 技术选型：语言分布雷达图帮助课题组选择技术栈
3. 人才发现：贡献者网络识别核心研发人员，为寻找合作者提供线索
4. 趋势预判：从镜像更新频率推断源仓库开发节奏，辅助科研选题
5. 思路提炼：自定义 Skill 自动提炼研究思路、技术迭代脉络、创新点

## 3. 落地效果
1. 在 GitLink 5 个真实仓库上验证通过
2. 生成 4 张可视化图表
3. 2 个自定义 Skill 可在 AI Agent 中一键触发，输出结构化报告

## 4. 可复现性说明
1. 环境：Python 3.9+，安装依赖 `pip install pandas matplotlib`
2. 数据采集：`bash scripts/fetch_data_v2.sh`
3. 数据分析：`python scripts/analyze_v2.py`
4. 可视化：`python scripts/visualize.py`
5. 一键运行：`bash run_all.sh`

## 附录
1. Skill 验证截图见 `skills/*/demo.png`