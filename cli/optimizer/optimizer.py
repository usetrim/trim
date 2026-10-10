#!/usr/bin/env python3
"""
Trim Deep Optimization bridge (Microsoft LLMLingua family).

Runs locally on the developer's machine. Invoked by the Go CLI via stdin/stdout JSON.
Never deploy this on Render: models need local RAM.

Engines:
  v1   - LLMLingua (perplexity / causal LM)
  long - LongLLMLingua (question-aware + context reorder)
  v2   - LLMLingua-2 (token classification, faster)

Model ids and device_map MUST come from the Go CLI request (env-backed). No invent.

Requires: pip install llmlingua  (auto-bootstrapped by `trim compress --deep` on first use)
"""

from __future__ import annotations

import json
import sys


def fail(message: str, code: int = 1) -> None:
    sys.stderr.write(message + "\n")
    sys.exit(code)


def main() -> None:
    try:
        raw = sys.stdin.read()
        if not raw.strip():
            fail("DEEP_OPT_EMPTY_STDIN")
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        fail(f"DEEP_OPT_INVALID_JSON:{exc}")

    text = payload.get("text") or ""
    question = payload.get("question") or ""
    raw_target = payload.get("target_token")
    if raw_target is None or raw_target == "":
        fail("DEEP_OPT_TARGET_TOKEN_REQUIRED")
    try:
        target_token = int(raw_target)
    except (TypeError, ValueError):
        fail(f"DEEP_OPT_TARGET_TOKEN_INVALID:{raw_target!r}")
    if target_token <= 0:
        fail("DEEP_OPT_TARGET_TOKEN_INVALID")
    raw_engine = payload.get("engine")
    if not raw_engine:
        fail("DEEP_OPT_ENGINE_REQUIRED")
    engine = str(raw_engine).lower().strip()
    rate = payload.get("rate")

    model_name = str(payload.get("model_name") or "").strip()
    device_map = str(payload.get("device_map") or "").strip()
    if not model_name:
        fail("DEEP_OPT_MODEL_NAME_REQUIRED")
    if not device_map:
        fail("DEEP_OPT_DEVICE_MAP_REQUIRED")

    force_tokens = payload.get("force_tokens")
    if force_tokens is None:
        force_tokens = []
    if not isinstance(force_tokens, list):
        fail("DEEP_OPT_FORCE_TOKENS_INVALID")
    use_llmlingua2 = bool(payload.get("use_llmlingua2"))

    if not text.strip():
        print(
            json.dumps(
                {
                    "compressed_prompt": "",
                    "origin_tokens": 0,
                    "compressed_tokens": 0,
                    "saving_rate": "0.0%",
                    "engine": engine,
                }
            )
        )
        return

    try:
        from llmlingua import PromptCompressor
    except ImportError:
        fail("DEEP_OPT_LLMLINGUA_MISSING")

    try:
        if engine == "v2":
            compressor = PromptCompressor(
                model_name=model_name,
                use_llmlingua2=use_llmlingua2,
                device_map=device_map,
            )
            # Microsoft LLMLingua: question/instruction are high-sensitivity; context is
            # what should be compressed hardest. Prefer structured call when question set.
            q = question.strip()
            kwargs = {"target_token": target_token}
            if force_tokens:
                kwargs["force_tokens"] = force_tokens
                # Official LLMLingua-2 examples pair force_tokens with drop_consecutive
                # and often force_reserve_digit so line numbers / versions survive.
                kwargs["drop_consecutive"] = True
                kwargs["force_reserve_digit"] = True
            if rate is not None:
                kwargs["rate"] = float(rate)
            if q:
                kwargs["question"] = q
                kwargs["instruction"] = ""
                try:
                    results = compressor.compress_prompt([text], **kwargs)
                except TypeError:
                    # Some LLMLingua-2 builds reject question=/instruction=/drop_consecutive.
                    fallback = {"target_token": target_token}
                    if force_tokens:
                        fallback["force_tokens"] = force_tokens
                    if rate is not None:
                        fallback["rate"] = float(rate)
                    results = compressor.compress_prompt(text, **fallback)
            else:
                try:
                    results = compressor.compress_prompt(text, **kwargs)
                except TypeError:
                    fallback = {"target_token": target_token}
                    if force_tokens:
                        fallback["force_tokens"] = force_tokens
                    if rate is not None:
                        fallback["rate"] = float(rate)
                    results = compressor.compress_prompt(text, **fallback)
        elif engine == "long":
            if not question.strip():
                fail("DEEP_OPT_LONG_QUESTION_REQUIRED")
            compressor = PromptCompressor(model_name=model_name, device_map=device_map)
            long_kwargs = {
                "question": question.strip(),
                "target_token": target_token,
                "reorder_context": "sort",
                "condition_compare": True,
                "rank_method": "longllmlingua",
            }
            if force_tokens:
                long_kwargs["force_tokens"] = force_tokens
                # Official LLMLingua DOCUMENT.md: pair force_tokens with digit reserve
                # so path segments / versions survive (coding-agent garble defense).
                long_kwargs["force_reserve_digit"] = True
                long_kwargs["drop_consecutive"] = True
            try:
                results = compressor.compress_prompt([text], **long_kwargs)
            except TypeError:
                long_kwargs.pop("force_reserve_digit", None)
                long_kwargs.pop("drop_consecutive", None)
                try:
                    results = compressor.compress_prompt([text], **long_kwargs)
                except TypeError:
                    long_kwargs.pop("force_tokens", None)
                    results = compressor.compress_prompt([text], **long_kwargs)
        else:
            compressor = PromptCompressor(model_name=model_name, device_map=device_map)
            q = question.strip()
            kwargs = {"target_token": target_token}
            if force_tokens:
                kwargs["force_tokens"] = force_tokens
                # Official Microsoft LLMLingua: force_reserve_digit + drop_consecutive
                # with force_tokens (same as v2 path) - protects digits in paths/versions.
                kwargs["force_reserve_digit"] = True
                kwargs["drop_consecutive"] = True
            if rate is not None:
                kwargs["rate"] = float(rate)
            if q:
                kwargs["question"] = q
                kwargs["instruction"] = ""
                try:
                    results = compressor.compress_prompt([text], **kwargs)
                except TypeError:
                    fallback = {"target_token": target_token}
                    if force_tokens:
                        fallback["force_tokens"] = force_tokens
                    if rate is not None:
                        fallback["rate"] = float(rate)
                    try:
                        results = compressor.compress_prompt(text, **fallback)
                    except TypeError:
                        fallback.pop("force_tokens", None)
                        results = compressor.compress_prompt(text, **fallback)
            else:
                try:
                    results = compressor.compress_prompt(text, **kwargs)
                except TypeError:
                    fallback = {"target_token": target_token}
                    if force_tokens:
                        fallback["force_tokens"] = force_tokens
                    if rate is not None:
                        fallback["rate"] = float(rate)
                    try:
                        results = compressor.compress_prompt(text, **fallback)
                    except TypeError:
                        fallback.pop("force_tokens", None)
                        results = compressor.compress_prompt(text, **fallback)
    except Exception as exc:  # noqa: BLE001 - surface model errors to CLI
        fail(f"DEEP_OPT_FAILED:{engine}:{exc}")

    origin = int(results.get("origin_tokens") or 0)
    compressed = int(results.get("compressed_tokens") or 0)
    saving = format_saving_rate(origin, compressed)

    out = {
        "compressed_prompt": results.get("compressed_prompt", ""),
        "origin_tokens": origin,
        "compressed_tokens": compressed,
        "saving_rate": saving,
        "engine": engine,
    }
    print(json.dumps(out))


def format_saving_rate(origin_tokens: int, compressed_tokens: int) -> str:
    """Percent of tokens removed: (origin - compressed) / origin * 100."""
    if origin_tokens <= 0:
        return "0.0%"
    rate = (origin_tokens - compressed_tokens) / float(origin_tokens) * 100.0
    return f"{rate:.1f}%"


if __name__ == "__main__":
    main()
