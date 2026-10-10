#!/usr/bin/env python3
"""Live multi-IDE attach E2E on a real machine using the real trim binary.

Proves (execution plane, not a mock HTTP server):
  1. Primary `trim start` owns the listen port and serves /health.
  2. Concurrent second/third `trim start` health-then-attach and exit 0 (no double-bind).
  3. Enforcer sidecar (`TRIM_AUTOSTART_ENFORCER=1`) attaches and stays up without stealing the port.
  4. Second Listen on the port is refused while primary is healthy.
  5. `trim stop` (pref-off / managed-off path) leaves /health down.

Chrome for attach comes from public.site_messages written into the CLI
last-known-good cache (same contract as offline daemon/login) - no invent bodies.

Run from repo root:
  python scripts/e2e_autostart_attach_live.py
"""
from __future__ import annotations

import json
import os
import shutil
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

import psycopg

ROOT = Path(__file__).resolve().parents[1]
CLI_DIR = ROOT / "cli"
TRIM_EXE = CLI_DIR / ("trim.exe" if os.name == "nt" else "trim")
CACHE_PATH = Path.home() / ".config" / "trim" / "cli-chrome-cache.json"

# site_messages codes required by chromeAutostartUsable + stop.
CHROME_CODES = (
    "DEFAULT_AUTO_START_WITH_IDE",
    "IDE_PROXY_HEALTH_URL",
    "IDE_PROXY_SHUTDOWN_URL",
    "CLI_PROXY_HEALTH_TIMEOUT_MS",
    "CLI_PROXY_SHUTDOWN_TIMEOUT_MS",
    "CLI_HTTP_SHUTDOWN_TIMEOUT_MS",
    "CLI_AUTOSTART_PREF_POLL_MS",
    "AUTOSTART_PREF_POLL_MIN_MS",
    "AUTOSTART_PREF_POLL_MAX_MS",
    "AUTOSTART_TIMEOUT_MIN_MS",
    "AUTOSTART_TIMEOUT_MAX_MS",
    "CLI_PROXY_FALLBACK_UNCOMPRESSED",
    "PROXY_ACTIVE_FILE_PROTECTION",
    "CLI_AUTOSTART_ENFORCER_STOPPED",
    "CLI_STOP_OK",
    "CLI_STOP_FAILED_FMT",
    "CLI_STOP_URL_MISSING",
)


def fail(msg: str) -> None:
    print(f"FAIL {msg}", file=sys.stderr)
    raise SystemExit(1)


def database_url() -> str:
    for env_path in (ROOT / "server" / ".env", CLI_DIR / ".env"):
        if not env_path.is_file():
            continue
        for line in env_path.read_text(encoding="utf-8").splitlines():
            if line.startswith("DATABASE_URL=") and not line.strip().startswith("#"):
                return line.split("=", 1)[1].strip().strip('"').strip("'")
    fail("DATABASE_URL missing (server/.env or cli/.env)")


def load_cli_env() -> dict[str, str]:
    env = os.environ.copy()
    env_path = CLI_DIR / ".env"
    if not env_path.is_file():
        fail("cli/.env missing")
    for line in env_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def fetch_site_messages() -> dict[str, str]:
    conn = psycopg.connect(database_url())
    try:
        cur = conn.cursor()
        cur.execute(
            "select code, body from site_messages where code = any(%s)",
            (list(CHROME_CODES),),
        )
        rows = {c: b for c, b in cur.fetchall()}
    finally:
        conn.close()
    missing = [c for c in CHROME_CODES if not str(rows.get(c, "")).strip()]
    if missing:
        fail(f"site_messages missing/empty: {missing}")
    return rows


def port_from_health_url(url: str) -> int:
    # http://127.0.0.1:8000/health
    try:
        from urllib.parse import urlparse

        p = urlparse(url)
        if p.port:
            return int(p.port)
    except Exception as e:  # noqa: BLE001
        fail(f"parse health url: {e}")
    fail(f"IDE_PROXY_HEALTH_URL has no port: {url!r}")


def write_chrome_cache(msgs: dict[str, str], backup: Path | None) -> Path | None:
    CACHE_PATH.parent.mkdir(parents=True, exist_ok=True)
    prev = None
    if CACHE_PATH.is_file():
        prev = CACHE_PATH.with_suffix(".json.e2e_bak")
        shutil.copy2(CACHE_PATH, prev)
        if backup is not None:
            pass
    # Go encoding/json uses exported field names (no struct tags on cliChrome).
    cache = {
        "DefaultAutoStartWithIDE": msgs["DEFAULT_AUTO_START_WITH_IDE"],
        "ProxyHealthURL": msgs["IDE_PROXY_HEALTH_URL"],
        "ProxyShutdownURL": msgs["IDE_PROXY_SHUTDOWN_URL"],
        "ProxyHealthTimeoutMs": msgs["CLI_PROXY_HEALTH_TIMEOUT_MS"],
        "ProxyShutdownTimeoutMs": msgs["CLI_PROXY_SHUTDOWN_TIMEOUT_MS"],
        "HTTPShutdownTimeoutMs": msgs["CLI_HTTP_SHUTDOWN_TIMEOUT_MS"],
        "AutostartPrefPollMs": msgs["CLI_AUTOSTART_PREF_POLL_MS"],
        "AutostartPrefPollMinMs": msgs["AUTOSTART_PREF_POLL_MIN_MS"],
        "AutostartPrefPollMaxMs": msgs["AUTOSTART_PREF_POLL_MAX_MS"],
        "AutostartTimeoutMinMs": msgs["AUTOSTART_TIMEOUT_MIN_MS"],
        "AutostartTimeoutMaxMs": msgs["AUTOSTART_TIMEOUT_MAX_MS"],
        "ProxyFallbackUncompressed": msgs["CLI_PROXY_FALLBACK_UNCOMPRESSED"],
        "ProxyActiveFileProtection": msgs["PROXY_ACTIVE_FILE_PROTECTION"],
        "AutostartEnforcerStopped": msgs["CLI_AUTOSTART_ENFORCER_STOPPED"],
        "StopOk": msgs["CLI_STOP_OK"],
        "StopFailedFmt": msgs["CLI_STOP_FAILED_FMT"],
        "StopURLMissing": msgs["CLI_STOP_URL_MISSING"],
    }
    CACHE_PATH.write_text(json.dumps(cache, indent=2), encoding="utf-8")
    return prev


def restore_chrome_cache(prev: Path | None) -> None:
    if prev and prev.is_file():
        shutil.move(str(prev), str(CACHE_PATH))
    elif CACHE_PATH.is_file():
        # We created the cache for this run only.
        CACHE_PATH.unlink(missing_ok=True)


def build_trim() -> None:
    print("building trim binary…")
    r = subprocess.run(
        ["go", "build", "-o", str(TRIM_EXE), "./cmd/trim"],
        cwd=CLI_DIR,
        capture_output=True,
        text=True,
    )
    if r.returncode != 0:
        fail(f"go build trim: {r.stderr or r.stdout}")


def proxy_health_ok(url: str, timeout: float = 1.0) -> bool:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as res:
            if res.status != 200:
                return False
            body = json.loads(res.read().decode("utf-8"))
            return body.get("status") == "ok" and body.get("proxy") == "trim"
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        return False


def wait_health(url: str, want_ok: bool, deadline_s: float) -> bool:
    end = time.time() + deadline_s
    while time.time() < end:
        if proxy_health_ok(url, 0.4) == want_ok:
            return True
        time.sleep(0.1)
    return False


def port_free(port: int) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.bind(("127.0.0.1", port))
        return True
    except OSError:
        return False
    finally:
        s.close()


def pick_free_port() -> int:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(("127.0.0.1", 0))
    port = int(s.getsockname()[1])
    s.close()
    return port


def main() -> None:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    msgs = fetch_site_messages()
    health_url = msgs["IDE_PROXY_HEALTH_URL"].strip()
    port = port_from_health_url(health_url)
    if not port_free(port):
        fail(f"port {port} already in use - stop existing Trim proxy first")

    build_trim()
    env = load_cli_env()
    env["TRIM_PORT"] = str(port)
    # Force fail-closed cache path: API may be unreachable on this machine.
    env["TRIM_API_BASE_URL"] = "http://127.0.0.1:9"

    prev_cache = write_chrome_cache(msgs, None)
    primary = None
    enforcer = None
    try:
        print(f"OK chrome seeded from site_messages → {CACHE_PATH}")
        print(f"OK using health={health_url} port={port}")

        primary = subprocess.Popen(
            [str(TRIM_EXE), "start", "-p", str(port)],
            cwd=CLI_DIR,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        if not wait_health(health_url, True, 12.0):
            out = primary.communicate(timeout=2)[0] if primary.poll() is not None else ""
            fail(f"primary trim start never became healthy (exit={primary.poll()} out={out[:800]!r})")
        print("OK primary trim start healthy (IDE #1 / first start)")

        # Multi-IDE attach: concurrent trim start without enforcer → exit 0, no double-bind.
        attach_clients = 3
        procs: list[subprocess.Popen[str]] = []
        for i in range(attach_clients):
            procs.append(
                subprocess.Popen(
                    [str(TRIM_EXE), "start", "-p", str(port)],
                    cwd=CLI_DIR,
                    env=env,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                )
            )
        attach_fail = 0
        for i, p in enumerate(procs, 1):
            try:
                code = p.wait(timeout=20)
            except subprocess.TimeoutExpired:
                p.kill()
                attach_fail += 1
                print(f"FAIL attach client {i} timed out", file=sys.stderr)
                continue
            if code != 0:
                attach_fail += 1
                out = (p.stdout.read() if p.stdout else "") or ""
                print(f"FAIL attach client {i} exit={code} out={out[:400]!r}", file=sys.stderr)
        if attach_fail:
            fail(f"{attach_fail}/{attach_clients} attach clients failed")
        if not proxy_health_ok(health_url):
            fail("proxy unhealthy after attach clients")
        print(f"OK {attach_clients} concurrent trim start attach clients exited 0 (no double-bind)")

        # Prove health-then-attach (not listen→addr-in-use→exit 0): different -p,
        # chrome health still primary. Must exit 0 and leave alt port unbound.
        alt_port = pick_free_port()
        if not port_free(alt_port):
            fail(f"alt port {alt_port} not free before attach-alt")
        alt = subprocess.run(
            [str(TRIM_EXE), "start", "-p", str(alt_port)],
            cwd=CLI_DIR,
            env=env,
            capture_output=True,
            text=True,
            timeout=20,
        )
        if alt.returncode != 0:
            fail(f"attach-alt exit={alt.returncode} out={(alt.stderr or alt.stdout)[:500]!r}")
        if not port_free(alt_port):
            fail(f"attach-alt bound :{alt_port} - did not health-then-attach to primary")
        if not proxy_health_ok(health_url):
            fail("primary unhealthy after attach-alt")
        print(f"OK health-then-attach: trim start -p {alt_port} exited 0 without binding alt port")

        # Enforcer sidecar (extension/daemon path).
        env_enf = dict(env)
        env_enf["TRIM_AUTOSTART_ENFORCER"] = "1"
        enforcer = subprocess.Popen(
            [str(TRIM_EXE), "start", "-p", str(port)],
            cwd=CLI_DIR,
            env=env_enf,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        time.sleep(1.5)
        if enforcer.poll() is not None:
            # Poll interval 0 ⇒ attach-and-exit is valid; poll>0 should stay.
            poll_ms = int(msgs["CLI_AUTOSTART_PREF_POLL_MS"])
            out = (enforcer.stdout.read() if enforcer.stdout else "") or ""
            if poll_ms == 0 and enforcer.returncode == 0:
                print("OK enforcer attached and exited (CLI_AUTOSTART_PREF_POLL_MS=0)")
            else:
                fail(f"enforcer exited early code={enforcer.returncode} poll_ms={poll_ms} out={out[:500]!r}")
        else:
            if not proxy_health_ok(health_url):
                fail("proxy unhealthy while enforcer sidecar running")
            print("OK enforcer sidecar attached and still running (TRIM_AUTOSTART_ENFORCER=1)")

        # Pref-off / stop path.
        stop = subprocess.run(
            [str(TRIM_EXE), "stop"],
            cwd=CLI_DIR,
            env=env,
            capture_output=True,
            text=True,
            timeout=20,
        )
        if stop.returncode != 0:
            fail(f"trim stop failed: {stop.stderr or stop.stdout}")
        if not wait_health(health_url, False, 8.0):
            fail("proxy still healthy after trim stop")
        print("OK trim stop shut down proxy (pref-off / managed-off path)")
        print("PASS e2e_autostart_attach_live")
    finally:
        if enforcer and enforcer.poll() is None:
            enforcer.terminate()
            try:
                enforcer.wait(timeout=5)
            except subprocess.TimeoutExpired:
                enforcer.kill()
        if primary and primary.poll() is None:
            primary.terminate()
            try:
                primary.wait(timeout=5)
            except subprocess.TimeoutExpired:
                primary.kill()
        restore_chrome_cache(prev_cache)


if __name__ == "__main__":
    main()
