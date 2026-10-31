#!/usr/bin/env python3
"""
Trim Deep Optimization bridge (Microsoft LLMLingua family).

Runs locally on the developer's machine. Invoked by the Go CLI via stdin/stdout JSON.
Never deploy this on Render: models need local RAM.

Engines:
  v1   - LLMLingua (perplexity / causal LM)
  long - LongLLMLingua (question-aware + context reorder)
  v2   - LLMLingua-2 (token classification, faster)

Compression control matches Microsoft HF demos:
  rate = after_tokens / before_tokens (keep-rate). Lower = more compression.

Model ids and device_map MUST come from the Go CLI request (env-backed). No invent.

Requires: pip install llmlingua
  Prefer: trim compress --bootstrap (green success banner when ready)
  Also auto-bootstrapped on first Deep use / live Deep when missing
"""

from __future__ import annotations

import json
import sys


def fail(message: str, code: int = 1) -> None:
    sys.stderr.write(message + "\n")
    sys.exit(code)


def parse_rate(payload: dict) -> float:
    raw = payload.get("rate")
    if raw is None or raw == "":
        fail("DEEP_OPT_RATE_REQUIRED")
    try:
        rate = float(raw)
    except (TypeError, ValueError):
        fail(f"DEEP_OPT_RATE_INVALID:{raw!r}")
    if rate <= 0 or rate > 1:
        fail("DEEP_OPT_RATE_INVALID")
    return rate


def billing_coding_kwargs(
    rate: float,
    force_tokens: list,
    force_reserve_digit: bool | None,
    drop_consecutive: bool | None,
    use_sentence_level_filter: bool | None,
) -> dict:
    """Billing-driven compress_prompt knobs only. Missing dials stay omitted (no invent)."""
    kwargs: dict = {"rate": rate}
    if force_tokens:
        kwargs["force_tokens"] = force_tokens
    if drop_consecutive is not None:
        kwargs["drop_consecutive"] = drop_consecutive
    if force_reserve_digit is not None:
        kwargs["force_reserve_digit"] = force_reserve_digit
    if use_sentence_level_filter is not None:
        kwargs["use_sentence_level_filter"] = use_sentence_level_filter
    return kwargs


def compress_prompt_compat(compressor, text: str, kwargs: dict, prefer_list: bool = False):
    """
    Call LLMLingua compress_prompt with graduated peel on TypeError.
    Keep billing coding dials as long as the installed library accepts them.
    Do not jump straight to rate-only (that previously dropped force_reserve_digit /
    drop_consecutive whenever question=/instruction= was rejected).
    """
    attempts: list[tuple[object, dict]] = []
    primary = [text] if prefer_list else text
    attempts.append((primary, dict(kwargs)))
    # Peel Microsoft question/instruction first (common reject on some LLMLingua-2 builds).
    no_qx = dict(kwargs)
    no_qx.pop("question", None)
    no_qx.pop("instruction", None)
    if no_qx != kwargs:
        attempts.append((text, no_qx))
    # Then peel coding dials only if the signature still rejects.
    no_dials = dict(no_qx)
    no_dials.pop("drop_consecutive", None)
    no_dials.pop("force_reserve_digit", None)
    no_dials.pop("use_sentence_level_filter", None)
    if no_dials != no_qx:
        attempts.append((text, no_dials))
    no_tokens = dict(no_dials)
    no_tokens.pop("force_tokens", None)
    if no_tokens != no_dials:
        attempts.append((text, no_tokens))

    last_err: TypeError | None = None
    for payload, kw in attempts:
        try:
            return compressor.compress_prompt(payload, **kw)
        except TypeError as exc:
            last_err = exc
            continue
    if last_err is not None:
        raise last_err
    fail("DEEP_OPT_COMPRESS_PROMPT_FAILED")


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
    rate = parse_rate(payload)
    raw_engine = payload.get("engine")
    if not raw_engine:
        fail("DEEP_OPT_ENGINE_REQUIRED")
    engine = str(raw_engine).lower().strip()

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
    # Billing-driven only (trim config sync). Missing keys = do not invent True.
    has_force_reserve = "force_reserve_digit" in payload
    has_drop_consecutive = "drop_consecutive" in payload
    has_sentence_filter = "use_sentence_level_filter" in payload
    force_reserve_digit = bool(payload.get("force_reserve_digit")) if has_force_reserve else None
    drop_consecutive = bool(payload.get("drop_consecutive")) if has_drop_consecutive else None
    use_sentence_level_filter = (
        bool(payload.get("use_sentence_level_filter")) if has_sentence_filter else None
    )

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
            # Microsoft LLMLingua-2 Space: rate is the primary knob (after/before).
            kwargs = billing_coding_kwargs(
                rate,
                force_tokens,
                force_reserve_digit,
                drop_consecutive,
                use_sentence_level_filter,
            )
            q = question.strip()
            if q:
                kwargs["question"] = q
                kwargs["instruction"] = ""
                results = compress_prompt_compat(compressor, text, kwargs, prefer_list=True)
            else:
                results = compress_prompt_compat(compressor, text, kwargs, prefer_list=False)
        elif engine == "long":
            if not question.strip():
                fail("DEEP_OPT_LONG_QUESTION_REQUIRED")
            compressor = PromptCompressor(model_name=model_name, device_map=device_map)
            long_kwargs = billing_coding_kwargs(
                rate,
                force_tokens,
                force_reserve_digit,
                drop_consecutive,
                use_sentence_level_filter,
            )
            long_kwargs.update(
                {
                    "question": question.strip(),
                    "reorder_context": "sort",
                    "condition_compare": True,
                    "rank_method": "longllmlingua",
                }
            )
            results = compress_prompt_compat(compressor, text, long_kwargs, prefer_list=True)
        else:
            compressor = PromptCompressor(model_name=model_name, device_map=device_map)
            kwargs = billing_coding_kwargs(
                rate,
                force_tokens,
                force_reserve_digit,
                drop_consecutive,
                use_sentence_level_filter,
            )
            q = question.strip()
            if q:
                kwargs["question"] = q
                kwargs["instruction"] = ""
                results = compress_prompt_compat(compressor, text, kwargs, prefer_list=True)
            else:
                results = compress_prompt_compat(compressor, text, kwargs, prefer_list=False)
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
