#!/usr/bin/env python3
"""Anthropic-compatible Messages API load tester.

Dependencies:
    pip install aiohttp

The script reads connection settings from environment variables:
    ANTHROPIC_BASE_URL
    ANTHROPIC_API_KEY
    ANTHROPIC_MODEL

Example:
    python anthropic_load_test.py --concurrency 20 --requests 500 --stream
"""

from __future__ import annotations

import argparse
import asyncio
import json
import math
import os
import random
import ssl
import statistics
import sys
import time
from collections import Counter
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

try:
    import aiohttp
except ImportError as exc:  # pragma: no cover
    raise SystemExit("缺少依赖 aiohttp，请先执行: pip install aiohttp") from exc


RETRYABLE_STATUS = {408, 429, 500, 502, 503, 504}


@dataclass(slots=True)
class RequestResult:
    request_id: int
    ok: bool
    status: int | None
    latency_ms: float
    ttft_ms: float | None
    input_tokens: int
    output_tokens: int
    attempts: int
    prompt_chars: int
    response_chars: int
    started_at: str
    error: str | None = None


class SmoothRateLimiter:
    """A simple global limiter that spaces request starts evenly."""

    def __init__(self, requests_per_second: float) -> None:
        self.interval = 1.0 / requests_per_second if requests_per_second > 0 else 0.0
        self._next_allowed = 0.0
        self._lock = asyncio.Lock()

    async def wait(self, deadline: float | None = None) -> bool:
        if self.interval <= 0:
            return deadline is None or asyncio.get_running_loop().time() < deadline
        loop = asyncio.get_running_loop()
        async with self._lock:
            now = loop.time()
            scheduled = max(now, self._next_allowed)
            if deadline is not None and scheduled >= deadline:
                return False
            self._next_allowed = scheduled + self.interval
        delay = scheduled - now
        if delay > 0:
            await asyncio.sleep(delay)
        return True


class WorkAllocator:
    def __init__(self, request_limit: int, duration_seconds: float) -> None:
        self.request_limit = request_limit
        self.deadline = (
            asyncio.get_running_loop().time() + duration_seconds
            if duration_seconds > 0
            else None
        )
        self._next_id = 0
        self._lock = asyncio.Lock()

    async def next_request_id(self) -> int | None:
        async with self._lock:
            now = asyncio.get_running_loop().time()
            if self.deadline is not None and now >= self.deadline:
                return None
            if self.deadline is None and self._next_id >= self.request_limit:
                return None
            request_id = self._next_id
            self._next_id += 1
            return request_id


DUMMY_SCENARIOS = (
    "将以下虚构客服工单分类为 billing、technical、account 或 other，并返回 JSON。",
    "阅读以下虚构系统事件，给出三条简短排障建议，并标注优先级。",
    "把以下虚构产品说明压缩成不超过五点的摘要，不要补充外部事实。",
    "检查以下虚构数据记录是否自洽，返回 valid、issues 和 corrected_values。",
    "根据以下虚构会议记录，提取 action_items、owners 和 deadlines。",
)


def build_dummy_prompt(request_id: int, target_chars: int, seed: int) -> str:
    rng = random.Random(seed + request_id * 1_000_003)
    scenario = rng.choice(DUMMY_SCENARIOS)
    ticket = {
        "request_id": request_id,
        "customer_id": f"dummy-{rng.randint(10000, 99999)}",
        "region": rng.choice(["ap-southeast", "us-west", "eu-central"]),
        "priority": rng.choice(["low", "medium", "high"]),
        "latency_ms": rng.randint(12, 4800),
        "error_rate": round(rng.random() * 0.25, 4),
        "retries": rng.randint(0, 8),
        "note": "所有数据均为压测生成的 dummy 数据。",
    }
    base = (
        f"{scenario}\n"
        f"数据：{json.dumps(ticket, ensure_ascii=False)}\n"
        "要求：答案结构清晰、内容简洁，且必须明确说明这是虚构数据。\n"
    )
    if target_chars <= len(base):
        return base[:target_chars]

    words = [
        "dummy", "synthetic", "load-test", "cache-bypass", "latency",
        "throughput", "token", "worker", "queue", "benchmark",
    ]
    filler_parts: list[str] = []
    current_length = len(base) + len("填充文本：")
    while current_length < target_chars:
        part = rng.choice(words) + " "
        filler_parts.append(part)
        current_length += len(part)
    return (base + "填充文本：" + "".join(filler_parts))[:target_chars]


def normalize_messages_url(base_url: str) -> str:
    base = base_url.rstrip("/")
    if base.endswith("/v1/messages"):
        return base
    if base.endswith("/v1"):
        return base + "/messages"
    return base + "/v1/messages"


def percentile(values: list[float], p: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * p
    low = math.floor(rank)
    high = math.ceil(rank)
    if low == high:
        return ordered[low]
    weight = rank - low
    return ordered[low] * (1 - weight) + ordered[high] * weight


def parse_retry_after(value: str | None) -> float | None:
    if not value:
        return None
    try:
        return max(0.0, float(value))
    except ValueError:
        return None


async def parse_streaming_response(
    response: aiohttp.ClientResponse,
    started_monotonic: float,
) -> tuple[str, int, int, float | None]:
    text_parts: list[str] = []
    input_tokens = 0
    output_tokens = 0
    ttft_ms: float | None = None

    async for raw_line in response.content:
        line = raw_line.decode("utf-8", errors="replace").strip()
        if not line.startswith("data:"):
            continue
        payload = line[5:].strip()
        if not payload or payload == "[DONE]":
            continue
        try:
            event = json.loads(payload)
        except json.JSONDecodeError:
            continue

        event_type = event.get("type")
        if event_type == "message_start":
            usage = event.get("message", {}).get("usage", {})
            input_tokens = int(usage.get("input_tokens") or 0)
            output_tokens = int(usage.get("output_tokens") or 0)
        elif event_type == "content_block_delta":
            delta = event.get("delta", {})
            text = delta.get("text")
            if isinstance(text, str):
                if ttft_ms is None:
                    ttft_ms = (time.perf_counter() - started_monotonic) * 1000
                text_parts.append(text)
        elif event_type == "message_delta":
            usage = event.get("usage", {})
            output_tokens = int(usage.get("output_tokens") or output_tokens)
        elif event_type == "error":
            error_obj = event.get("error", {})
            raise RuntimeError(
                f"stream_error: {error_obj.get('type', 'unknown')}: "
                f"{error_obj.get('message', 'unknown error')}"
            )

    return "".join(text_parts), input_tokens, output_tokens, ttft_ms


async def parse_json_response(
    response: aiohttp.ClientResponse,
) -> tuple[str, int, int]:
    data = await response.json(content_type=None)
    content = data.get("content", []) if isinstance(data, dict) else []
    text_parts: list[str] = []
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and isinstance(block.get("text"), str):
                text_parts.append(block["text"])
    usage = data.get("usage", {}) if isinstance(data, dict) else {}
    return (
        "".join(text_parts),
        int(usage.get("input_tokens") or 0),
        int(usage.get("output_tokens") or 0),
    )


async def execute_request(
    *,
    session: aiohttp.ClientSession,
    url: str,
    headers: dict[str, str],
    model: str,
    request_id: int,
    prompt: str,
    max_tokens: int,
    temperature: float,
    stream: bool,
    retries: int,
    retry_base_seconds: float,
) -> RequestResult:
    started_iso = datetime.now(timezone.utc).isoformat()
    overall_start = time.perf_counter()
    last_error: str | None = None
    last_status: int | None = None

    payload: dict[str, Any] = {
        "model": model,
        "max_tokens": max_tokens,
        "temperature": temperature,
        "messages": [{"role": "user", "content": prompt}],
        "stream": stream,
    }

    for attempt in range(1, retries + 2):
        attempt_start = time.perf_counter()
        try:
            async with session.post(url, headers=headers, json=payload) as response:
                last_status = response.status
                if not 200 <= response.status < 300:
                    error_text = (await response.text())[:1000]
                    last_error = f"HTTP {response.status}: {error_text}"
                    if response.status in RETRYABLE_STATUS and attempt <= retries:
                        retry_after = parse_retry_after(response.headers.get("Retry-After"))
                        delay = retry_after if retry_after is not None else retry_base_seconds * (2 ** (attempt - 1))
                        await asyncio.sleep(delay + random.random() * 0.1)
                        continue
                    break

                if stream:
                    response_text, input_tokens, output_tokens, ttft_ms = await parse_streaming_response(
                        response, attempt_start
                    )
                else:
                    response_text, input_tokens, output_tokens = await parse_json_response(response)
                    ttft_ms = None

                return RequestResult(
                    request_id=request_id,
                    ok=True,
                    status=response.status,
                    latency_ms=(time.perf_counter() - overall_start) * 1000,
                    ttft_ms=ttft_ms,
                    input_tokens=input_tokens,
                    output_tokens=output_tokens,
                    attempts=attempt,
                    prompt_chars=len(prompt),
                    response_chars=len(response_text),
                    started_at=started_iso,
                )
        except (aiohttp.ClientError, asyncio.TimeoutError, ssl.SSLError, RuntimeError) as exc:
            last_error = f"{type(exc).__name__}: {exc}"
            if attempt <= retries:
                await asyncio.sleep(retry_base_seconds * (2 ** (attempt - 1)) + random.random() * 0.1)
                continue
            break
        except Exception as exc:  # Keep one unexpected failure from killing all workers.
            last_error = f"unexpected {type(exc).__name__}: {exc}"
            break

    return RequestResult(
        request_id=request_id,
        ok=False,
        status=last_status,
        latency_ms=(time.perf_counter() - overall_start) * 1000,
        ttft_ms=None,
        input_tokens=0,
        output_tokens=0,
        attempts=min(retries + 1, attempt),
        prompt_chars=len(prompt),
        response_chars=0,
        started_at=started_iso,
        error=last_error or "unknown error",
    )


async def run_load_test(args: argparse.Namespace) -> tuple[list[RequestResult], float]:
    api_key = os.getenv("ANTHROPIC_API_KEY", "").strip()
    base_url = (args.base_url or os.getenv("ANTHROPIC_BASE_URL", "")).strip()
    model = (args.model or os.getenv("ANTHROPIC_MODEL", "")).strip()

    missing = [
        name
        for name, value in (
            ("ANTHROPIC_BASE_URL", base_url),
            ("ANTHROPIC_API_KEY", api_key),
            ("ANTHROPIC_MODEL", model),
        )
        if not value
    ]
    if missing:
        raise SystemExit("缺少配置: " + ", ".join(missing))

    parsed = urlparse(base_url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise SystemExit(f"ANTHROPIC_BASE_URL 格式无效: {base_url!r}")

    url = normalize_messages_url(base_url)
    headers = {
        "x-api-key": api_key,
        "anthropic-version": args.anthropic_version,
        "content-type": "application/json",
        "user-agent": "dummy-anthropic-load-test/1.0",
    }

    ssl_context: ssl.SSLContext | bool = False if args.insecure else True

    connector = aiohttp.TCPConnector(
        limit=args.concurrency,
        limit_per_host=args.concurrency,
        ssl=ssl_context,
        ttl_dns_cache=300,
        enable_cleanup_closed=True,
    )
    timeout = aiohttp.ClientTimeout(total=args.timeout, connect=args.connect_timeout)

    print(
        json.dumps(
            {
                "endpoint": url,
                "model": model,
                "concurrency": args.concurrency,
                "requests": None if args.duration > 0 else args.requests,
                "duration_seconds": args.duration if args.duration > 0 else None,
                "target_rps": args.rps if args.rps > 0 else "unlimited",
                "stream": args.stream,
                "prompt_chars": args.prompt_chars,
                "max_tokens": args.max_tokens,
                "retries": args.retries,
                "tls_verification": not args.insecure,
            },
            ensure_ascii=False,
            indent=2,
        )
    )

    results: list[RequestResult] = []
    results_lock = asyncio.Lock()

    async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
        # Warm-up calls are intentionally excluded from the final metrics.
        for warmup_id in range(args.warmup):
            prompt = build_dummy_prompt(-(warmup_id + 1), args.prompt_chars, args.seed)
            warmup_result = await execute_request(
                session=session,
                url=url,
                headers=headers,
                model=model,
                request_id=-(warmup_id + 1),
                prompt=prompt,
                max_tokens=args.max_tokens,
                temperature=args.temperature,
                stream=args.stream,
                retries=args.retries,
                retry_base_seconds=args.retry_base_seconds,
            )
            if not warmup_result.ok:
                print(f"警告: warm-up 失败: {warmup_result.error}", file=sys.stderr)

        # Start the allocator and wall clock after warm-up so duration and RPS
        # measurements only cover the actual load-test window.
        allocator = WorkAllocator(args.requests, args.duration)
        rate_limiter = SmoothRateLimiter(args.rps)
        load_start = time.perf_counter()

        async def worker(worker_id: int) -> None:
            while True:
                request_id = await allocator.next_request_id()
                if request_id is None:
                    return
                if not await rate_limiter.wait(allocator.deadline):
                    return
                prompt = build_dummy_prompt(request_id, args.prompt_chars, args.seed)
                result = await execute_request(
                    session=session,
                    url=url,
                    headers=headers,
                    model=model,
                    request_id=request_id,
                    prompt=prompt,
                    max_tokens=args.max_tokens,
                    temperature=args.temperature,
                    stream=args.stream,
                    retries=args.retries,
                    retry_base_seconds=args.retry_base_seconds,
                )
                async with results_lock:
                    results.append(result)
                    completed = len(results)
                    if args.progress_every > 0 and completed % args.progress_every == 0:
                        success = sum(item.ok for item in results)
                        print(
                            f"progress completed={completed} success={success} "
                            f"failed={completed - success}",
                            file=sys.stderr,
                        )

        await asyncio.gather(*(worker(i) for i in range(args.concurrency)))
        load_wall_seconds = time.perf_counter() - load_start

    return results, load_wall_seconds


def make_summary(results: list[RequestResult], wall_seconds: float) -> dict[str, Any]:
    successes = [r for r in results if r.ok]
    failures = [r for r in results if not r.ok]
    latencies = [r.latency_ms for r in successes]
    ttfts = [r.ttft_ms for r in successes if r.ttft_ms is not None]
    total_input_tokens = sum(r.input_tokens for r in successes)
    total_output_tokens = sum(r.output_tokens for r in successes)

    def rounded(value: float | None) -> float | None:
        return round(value, 2) if value is not None else None

    status_counts = Counter(str(r.status) if r.status is not None else "network_error" for r in results)
    error_counts = Counter((r.error or "unknown").split(":", 1)[0] for r in failures)

    return {
        "wall_seconds": round(wall_seconds, 3),
        "completed": len(results),
        "success": len(successes),
        "failed": len(failures),
        "success_rate_pct": round((len(successes) / len(results) * 100), 3) if results else 0.0,
        "throughput_rps": round(len(results) / wall_seconds, 3) if wall_seconds > 0 else 0.0,
        "successful_rps": round(len(successes) / wall_seconds, 3) if wall_seconds > 0 else 0.0,
        "latency_ms": {
            "min": rounded(min(latencies)) if latencies else None,
            "mean": rounded(statistics.fmean(latencies)) if latencies else None,
            "p50": rounded(percentile(latencies, 0.50)),
            "p90": rounded(percentile(latencies, 0.90)),
            "p95": rounded(percentile(latencies, 0.95)),
            "p99": rounded(percentile(latencies, 0.99)),
            "max": rounded(max(latencies)) if latencies else None,
        },
        "ttft_ms": {
            "mean": rounded(statistics.fmean(ttfts)) if ttfts else None,
            "p50": rounded(percentile(ttfts, 0.50)),
            "p95": rounded(percentile(ttfts, 0.95)),
            "p99": rounded(percentile(ttfts, 0.99)),
        },
        "tokens": {
            "input_total": total_input_tokens,
            "output_total": total_output_tokens,
            "output_tokens_per_second": round(total_output_tokens / wall_seconds, 3) if wall_seconds > 0 else 0.0,
            "avg_output_per_success": round(total_output_tokens / len(successes), 2) if successes else 0.0,
        },
        "attempts_total": sum(r.attempts for r in results),
        "status_counts": dict(status_counts),
        "error_type_counts": dict(error_counts),
    }


def write_jsonl(path: Path, results: list[RequestResult]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        for result in sorted(results, key=lambda item: item.request_id):
            handle.write(json.dumps(asdict(result), ensure_ascii=False) + "\n")


def positive_int(value: str) -> int:
    parsed = int(value)
    if parsed <= 0:
        raise argparse.ArgumentTypeError("必须大于 0")
    return parsed


def non_negative_int(value: str) -> int:
    parsed = int(value)
    if parsed < 0:
        raise argparse.ArgumentTypeError("不能小于 0")
    return parsed


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="对 Anthropic-compatible /v1/messages 接口进行异步并发压测。"
    )
    parser.add_argument("--base-url", help="覆盖 ANTHROPIC_BASE_URL")
    parser.add_argument("--model", help="覆盖 ANTHROPIC_MODEL")
    parser.add_argument("-c", "--concurrency", type=positive_int, default=10, help="并发 worker 数，默认 10")
    parser.add_argument("-n", "--requests", type=positive_int, default=100, help="总请求数，默认 100")
    parser.add_argument("-d", "--duration", type=float, default=0.0, help="持续压测秒数；大于 0 时忽略 --requests")
    parser.add_argument("--rps", type=float, default=0.0, help="限制请求启动速率；0 表示不限制")
    parser.add_argument("--warmup", type=non_negative_int, default=2, help="正式统计前的预热请求数，默认 2")
    parser.add_argument("--prompt-chars", type=positive_int, default=512, help="每条 dummy prompt 的目标字符数")
    parser.add_argument("--max-tokens", type=positive_int, default=128, help="单次最大输出 token 数")
    parser.add_argument("--temperature", type=float, default=0.2, help="采样温度")
    parser.add_argument("--stream", action="store_true", help="使用 SSE 流式响应并统计 TTFT")
    parser.add_argument("--timeout", type=float, default=120.0, help="单次请求总超时秒数")
    parser.add_argument("--connect-timeout", type=float, default=10.0, help="连接超时秒数")
    parser.add_argument("--retries", type=non_negative_int, default=0, help="失败重试次数；压测默认不重试")
    parser.add_argument("--retry-base-seconds", type=float, default=0.5, help="指数退避基础秒数")
    parser.add_argument("--anthropic-version", default="2023-06-01", help="anthropic-version 请求头")
    parser.add_argument("--insecure", action="store_true", help="关闭 TLS 证书校验，仅用于受控测试环境")
    parser.add_argument("--seed", type=int, default=20260722, help="dummy 数据随机种子")
    parser.add_argument("--output", default="load_test_results.jsonl", help="请求明细 JSONL 输出路径")
    parser.add_argument("--summary-output", default="load_test_summary.json", help="汇总 JSON 输出路径")
    parser.add_argument("--progress-every", type=non_negative_int, default=50, help="每完成 N 条打印进度；0 表示关闭")
    return parser


def validate_args(args: argparse.Namespace) -> None:
    for name in ("duration", "rps", "timeout", "connect_timeout", "retry_base_seconds"):
        if getattr(args, name) < 0:
            raise SystemExit(f"--{name.replace('_', '-')} 不能小于 0")
    if not 0 <= args.temperature <= 2:
        raise SystemExit("--temperature 必须位于 0 到 2 之间")


async def async_main(args: argparse.Namespace) -> int:
    results, wall_seconds = await run_load_test(args)
    summary = make_summary(results, wall_seconds)

    output_path = Path(args.output)
    summary_path = Path(args.summary_output)
    write_jsonl(output_path, results)
    summary_path.parent.mkdir(parents=True, exist_ok=True)
    summary_path.write_text(
        json.dumps(summary, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    print("\n=== Load Test Summary ===")
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    print(f"\n明细: {output_path.resolve()}")
    print(f"汇总: {summary_path.resolve()}")

    return 0 if summary["failed"] == 0 else 2


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    validate_args(args)
    try:
        return asyncio.run(async_main(args))
    except KeyboardInterrupt:
        print("\n压测已由用户中止。", file=sys.stderr)
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
