"""llm.py — 可选 LLM 客户端（DeepSeek，OpenAI 兼容），用于创新启发生成 idea。

无 DEEPSEEK_API_KEY 时 chat() 返回 None（调用方降级，不中断）。纯 stdlib，无新依赖。

环境变量：
  DEEPSEEK_API_KEY  (或 LLM_API_KEY)   — 必填才启用
  DEEPSEEK_BASE_URL (或 LLM_BASE_URL)  — 默认 https://api.deepseek.com
  DEEPSEEK_MODEL    (或 LLM_MODEL)     — 默认 deepseek-chat

冒烟：python llm.py "说一句你好"
"""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

DEFAULT_BASE = "https://api.deepseek.com"
DEFAULT_MODEL = "deepseek-chat"
TIMEOUT = 60


def _api_key() -> str:
    return os.environ.get("DEEPSEEK_API_KEY") or os.environ.get("LLM_API_KEY") or ""


def _base_url() -> str:
    return (os.environ.get("DEEPSEEK_BASE_URL") or os.environ.get("LLM_BASE_URL") or DEFAULT_BASE).rstrip("/")


def _model() -> str:
    return os.environ.get("DEEPSEEK_MODEL") or os.environ.get("LLM_MODEL") or DEFAULT_MODEL


def available() -> bool:
    """是否配置了 key（链路据此决定是否调用 LLM）。"""
    return bool(_api_key())


def chat(prompt: str, system: str | None = None, model: str | None = None,
         max_tokens: int = 900, temperature: float = 0.7) -> str | None:
    """调用 DeepSeek（OpenAI 兼容 /chat/completions）。

    返回助手回复文本；无 key、网络/解析出错则返回 None（调用方应降级）。
    """
    key = _api_key()
    if not key:
        return None
    messages = []
    if system:
        messages.append({"role": "system", "content": system})
    messages.append({"role": "user", "content": prompt})
    body = {
        "model": model or _model(),
        "messages": messages,
        "max_tokens": max_tokens,
        "temperature": temperature,
        "stream": False,
    }
    req = urllib.request.Request(
        _base_url() + "/chat/completions",
        data=json.dumps(body).encode("utf-8"),
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as r:
            data = json.loads(r.read().decode("utf-8"))
        choices = data.get("choices") or []
        if choices:
            return (choices[0].get("message") or {}).get("content")
    except (urllib.error.URLError, urllib.error.HTTPError, TimeoutError, ValueError) as e:
        sys.stderr.write(f"[llm] 调用失败（链路将降级）: {e!r}\n")
    return None


def main() -> None:
    import argparse
    ap = argparse.ArgumentParser(description="DeepSeek LLM 冒烟测试")
    ap.add_argument("prompt", nargs="?", default="说一句你好", help="prompt")
    ap.add_argument("--system", default=None)
    args = ap.parse_args()
    if not available():
        print("(no DEEPSEEK_API_KEY / LLM_API_KEY — chat 不可用；链路会自动降级，确定性产出照常)")
        return
    out = chat(args.prompt, system=args.system)
    print(out or "(LLM 返回空)")


if __name__ == "__main__":
    main()
