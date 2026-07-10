"""topics.py — 科研主题/技术关键词词典（S2 知识图谱与 S4 协作匹配共享）。

采用「关键词词典 + 字符串匹配」方式抽取主题，零 NLP/分词依赖，结果确定可复现。
词典可扩展；覆盖 GitLink 上常见科研方向（CV/NLP/RL/DL/系统/安全/科学计算等）。
"""
from __future__ import annotations

import re
from collections import Counter
from typing import Iterable

# 主题 → 触发关键词（中英文）。小写匹配。
TOPIC_KEYWORDS: dict[str, tuple[str, ...]] = {
    "machine_learning": ("机器学习", "machine learning", "监督学习", "无监督学习",
                         "supervised", "unsupervised", "scikit-learn", "sklearn",
                         "特征工程", "feature engineering", "generalization"),
    "computer_vision": ("目标检测", "图像分类", "语义分割", "实例分割", "目标跟踪",
                        "object detection", "image classification", "semantic segmentation",
                        "instance segmentation", "object tracking", "yolo", "resnet", "cnn",
                        "图像识别", "ocr", "人脸识别", "visual", "vision"),
    "nlp": ("自然语言处理", "文本分类", "机器翻译", "问答", "命名实体",
            "nlp", "text classification", "machine translation", "transformer", "bert",
            "gpt", "llm", "大模型", "大语言模型", "预训练", "pretrain", "分词", "tokeniz"),
    "generative_ai": ("生成式", "aigc", "扩散模型", "生成模型", "文生图", "多模态",
                      "generative", "diffusion", "gan", "vae", "multimodal", "clip",
                      "对话", "chatgpt", "chat", "instruction tun"),
    "reinforcement_learning": ("强化学习", "多智能体", "决策",
                               "reinforcement learning", "multi-agent", "ppo", "dqn",
                               "q-learning", "reward", "policy gradient"),
    "deep_learning": ("深度学习", "神经网络", "训练", "推理", "微调",
                      "deep learning", "neural network", "pytorch", "tensorflow",
                      "mindspore", "paddle", "paddlepaddle", "inference", "fine-tun",
                      "backbone", "checkpoint"),
    "graph_learning": ("图神经网络", "图表示学习", "知识图谱",
                       "graph neural", "gnn", "graph convolution", "gcn", "graphsage",
                       "knowledge graph", "图嵌入", "graph embed"),
    "federated_learning": ("联邦学习", "隐私保护", "分布式训练",
                           "federated", "privacy", "distributed training"),
    "speech": ("语音识别", "语音合成", "声纹", "语音",
               "speech", "asr", "tts", "speaker", "voice", "声学"),
    "scientific_computing": ("科学计算", "数值模拟", "高性能计算", "并行计算",
                             "numerical", "simulation", "hpc", "parallel", "cuda", "gpu",
                             "有限元", "偏微分"),
    "autonomous_systems": ("自动驾驶", "机器人", "感知", "导航", "slam",
                           "autonomous", "robotics", "robot", "self-driving", "planning"),
    "bioinformatics": ("生物信息", "蛋白质", "基因", "分子",
                       "bioinformatic", "genomic", "protein", "molecular", "drug"),
    "time_series": ("时序", "时间序列", "预测", "序列建模",
                    "time series", "time-series", "forecasting", "temporal"),
    "devops": ("ci/cd", "devops", "pipeline", "容器", "编排",
               "docker", "kubernetes", "k8s", "jenkins", "自动化部署", "helm"),
    "database": ("数据库", "存储", "索引",
                 "database", "sql", "nosql", "storage", "index"),
    "security": ("安全", "漏洞", "加密", "隐私",
                 "security", "vulnerability", "crypto", "privacy", "attack"),
    "data_mining": ("数据挖掘", "推荐系统", "聚类", "分类",
                    "data mining", "recommender", "clustering", "classification", "tf-idf"),
}

# 编程语言关键词（用于 S4 语言匹配）
LANGUAGE_KEYWORDS: tuple[str, ...] = (
    "python", "go", "golang", "c++", "cpp", "c#", "java", "rust", "javascript",
    "typescript", "julia", "r", "matlab", "scala", "swift", "kotlin", "cuda",
)

_NON_ALNUM = re.compile(r"[^\w一-鿿+#]+")


def _normalize(text: str) -> str:
    return (text or "").lower()


def extract_topics(text: str) -> list[str]:
    """从一段文本中抽取命中的主题列表（去重，保序）。"""
    t = _normalize(text)
    if not t:
        return []
    hit = []
    for topic, kws in TOPIC_KEYWORDS.items():
        for kw in kws:
            if _normalize(kw) in t:
                hit.append(topic)
                break
    return hit


def extract_languages(text: str) -> list[str]:
    """从文本中抽取命中的编程语言（归一化别名，如 golang→go）。"""
    t = _normalize(text)
    if not t:
        return []
    alias = {"golang": "go", "cpp": "c++", "c#": "c#", "ts": "typescript"}
    out, seen = [], set()
    # 按非字母数字分割后逐 token 比对，避免 'go' 误命中 'google'
    tokens = set(_NON_ALNUM.sub(" ", t).split())
    for kw in LANGUAGE_KEYWORDS:
        norm = _normalize(kw)
        if norm in tokens and norm not in seen:
            seen.add(norm)
            out.append(alias.get(norm, norm))
    return out


def topic_counter(texts: Iterable[str]) -> Counter:
    """对多段文本累计主题词频，用于热点排序（S2）。"""
    c: Counter = Counter()
    for t in texts:
        for topic in extract_topics(t):
            c[topic] += 1
    return c
