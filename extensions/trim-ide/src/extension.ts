import { clearInterval, clearTimeout, setInterval, setTimeout } from "node:timers";
import { createHash } from "node:crypto";
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import * as vscode from "vscode";
import {
  clientVersionHeader as formatClientVersion,
  emptyCounters,
  joinApi,
  normalizeCounters,
  parseApiBase as parseApiBasePure,
  pendingTotal,
  userAgentHeader as formatUserAgent,
  type CounterSnapshot,
} from "./util";

const SECRET_KEY = "trim.apiKey";
/** Pending telemetry counters survive reload/crash until a successful flush. */
const COUNTERS_STATE_KEY = "trim.pendingCounters";

type Counters = CounterSnapshot;

/** Stable hashed machine id for X-Hardware-UUID (IDE-specific; not the same as CLI). */
function ideHardwareUUID(): string {
  const raw = `${vscode.env.machineId}|trim-ide`;
  return createHash("sha256").update(raw).digest("hex");
}

function loadPersistedCounters(context: vscode.ExtensionContext): Counters {
  return normalizeCounters(context.globalState.get<Partial<Counters>>(COUNTERS_STATE_KEY));
}

async function persistCounters(
  context: vscode.ExtensionContext,
  counters: Counters,
): Promise<void> {
  if (pendingTotal(counters) === 0) {
    await context.globalState.update(COUNTERS_STATE_KEY, undefined);
    return;
  }
  await context.globalState.update(COUNTERS_STATE_KEY, {
    tabShown: counters.tabShown,
    tabAccepted: counters.tabAccepted,
    linesAdded: counters.linesAdded,
    linesDeleted: counters.linesDeleted,
  });
}

/** Required site_messages codes from GET /api/v1/public/ide-chrome (fail-closed). */
const REQUIRED_CHROME = [
  "IDE_STATUS_TOOLTIP",
  "IDE_STATUS_IDLE",
  "IDE_STATUS_PENDING_FMT",
  "IDE_API_KEY_TITLE",
  "IDE_API_KEY_PROMPT",
  "IDE_API_KEY_SAVED",
  "IDE_API_KEY_CLEARED",
  "IDE_TELEMETRY_FLUSHED",
  "IDE_TELEMETRY_FAILED_FMT",
  "IDE_TELEMETRY_NETWORK_FMT",
  "IDE_EVENT_MODE",
  "IDE_EVENT_STATUS",
  "IDE_EVENT_MODEL",
  "IDE_CMD_COPY_HARDWARE_ID",
  "IDE_HARDWARE_ID_COPIED",
  "IDE_AUTOSTART_ENSURING",
  "IDE_AUTOSTART_PROXY_OK",
  "IDE_AUTOSTART_PROXY_STARTED",
  "IDE_AUTOSTART_PROXY_FAILED_FMT",
  "IDE_AUTOSTART_SKIPPED_OFF",
  "IDE_AUTOSTART_SKIPPED_UNSET",
  "IDE_AUTOSTART_CLI_MISSING",
  "IDE_PROXY_HEALTH_URL",
  "IDE_AUTOSTART_SETTLE_MS",
  "IDE_AUTOSTART_STOP_ON_QUIT",
  "IDE_PROXY_SHUTDOWN_URL",
  "IDE_AUTOSTART_STATUS_OK",
  "IDE_AUTOSTART_STATUS_FAILED",
  "IDE_AUTOSTART_WARN_FAILED",
  "IDE_AUTOSTART_SKIPPED_LOCAL_OFF",
  "IDE_AUTOSTART_SKIPPED_MANAGED",
  "IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS",
  "IDE_AUTOSTART_PREF_POLL_MS",
  "AUTOSTART_PREF_POLL_MIN_MS",
  "AUTOSTART_PREF_POLL_MAX_MS",
  "AUTOSTART_TIMEOUT_MIN_MS",
  "AUTOSTART_TIMEOUT_MAX_MS",
  "IDE_AUTOSTART_HEALTH_AFTER_START_FAILED",
  "IDE_HTTP_TIMEOUT_MIN_SEC",
  "IDE_HTTP_TIMEOUT_MAX_SEC",
  "IDE_AUTO_FLUSH_MIN_SEC",
  "IDE_AUTO_FLUSH_MAX_SEC",
  "IDE_AUTO_FLUSH_MIN_ENABLED_SEC",
  "IDE_HTTP_TIMEOUT_INVALID",
  "IDE_AUTO_FLUSH_INVALID",
  "IDE_CONFIG_TRACK_EDITS_MISSING",
  "IDE_CONFIG_MIN_LINES_INVALID",
  "IDE_CONFIG_API_URL_EMPTY",
  "IDE_CONFIG_API_URL_INVALID",
  "DEFAULT_AUTO_START_WITH_IDE",
  "IDE_QUOTA_EXHAUSTED_TITLE",
  "IDE_QUOTA_EXHAUSTED_BODY",
  "IDE_QUOTA_UPGRADE_ACTION",
] as const;

type ChromeCode = (typeof REQUIRED_CHROME)[number];
type IdeChrome = Partial<Record<ChromeCode, string>>;

type TrimConfig = {
  apiBase: string;
  httpTimeoutMs: number;
  autoFlushSeconds: number;
  trackDocumentEdits: boolean;
  minLinesForAiHeuristic: number;
};

let output: vscode.OutputChannel | undefined;
let chrome: IdeChrome = {};
let extensionVersion = "0.0.0";
let flushInFlight: Promise<boolean> | undefined;
/** One Upgrade toast per extension host session when cloud credits are exhausted. */
let quotaExhaustedNotified = false;
/** Kept for best-effort flush on deactivate. */
let runtime:
  | {
      context: vscode.ExtensionContext;
      counters: Counters;
    }
  | undefined;

function log(line: string): void {
  output?.appendLine(`[${new Date().toISOString()}] ${line}`);
}

function readConfig(): TrimConfig | undefined {
  const cfg = vscode.workspace.getConfiguration("trim");
  const rawUrl = cfg.get<string>("apiUrl");
  const timeoutSec = cfg.get<number>("httpTimeoutSec");
  const autoFlushSeconds = cfg.get<number>("autoFlushSeconds");
  const trackDocumentEdits = cfg.get<boolean>("trackDocumentEdits");
  const minLinesForAiHeuristic = cfg.get<number>("minLinesForAiHeuristic");

  // Structural checks. Prefer DB chrome copy when last-known-good cache already loaded.
  if (timeoutSec === undefined || !Number.isFinite(timeoutSec) || timeoutSec < 1) {
    if (chrome.IDE_HTTP_TIMEOUT_INVALID) {
      log(chrome.IDE_HTTP_TIMEOUT_INVALID);
    }
    return undefined;
  }
  if (
    autoFlushSeconds === undefined ||
    !Number.isFinite(autoFlushSeconds) ||
    autoFlushSeconds < 0
  ) {
    if (chrome.IDE_AUTO_FLUSH_INVALID) {
      log(chrome.IDE_AUTO_FLUSH_INVALID);
    }
    return undefined;
  }
  if (trackDocumentEdits === undefined) {
    if (chrome.IDE_CONFIG_TRACK_EDITS_MISSING) {
      log(chrome.IDE_CONFIG_TRACK_EDITS_MISSING);
    }
    return undefined;
  }
  if (
    minLinesForAiHeuristic === undefined ||
    !Number.isFinite(minLinesForAiHeuristic) ||
    minLinesForAiHeuristic < 0
  ) {
    if (chrome.IDE_CONFIG_MIN_LINES_INVALID) {
      log(chrome.IDE_CONFIG_MIN_LINES_INVALID);
    }
    return undefined;
  }

  const apiBase = parseApiBase(rawUrl ?? "");
  return {
    apiBase,
    httpTimeoutMs: timeoutSec * 1000,
    autoFlushSeconds,
    trackDocumentEdits,
    minLinesForAiHeuristic,
  };
}

/** Validate VS Code settings against DB-driven ide-chrome bounds. Fail-closed. */
function configWithinChromeBounds(cfg: TrimConfig): boolean {
  const minHttp = Number.parseInt((chrome.IDE_HTTP_TIMEOUT_MIN_SEC || "").trim(), 10);
  const maxHttp = Number.parseInt((chrome.IDE_HTTP_TIMEOUT_MAX_SEC || "").trim(), 10);
  const minFlush = Number.parseInt((chrome.IDE_AUTO_FLUSH_MIN_SEC || "").trim(), 10);
  const maxFlush = Number.parseInt((chrome.IDE_AUTO_FLUSH_MAX_SEC || "").trim(), 10);
  const minFlushEnabled = Number.parseInt((chrome.IDE_AUTO_FLUSH_MIN_ENABLED_SEC || "").trim(), 10);
  const httpOk =
    Number.isFinite(minHttp) &&
    Number.isFinite(maxHttp) &&
    minHttp >= 1 &&
    maxHttp >= 1 &&
    minHttp <= maxHttp;
  const flushOk =
    Number.isFinite(minFlush) &&
    Number.isFinite(maxFlush) &&
    Number.isFinite(minFlushEnabled) &&
    minFlush >= 0 &&
    maxFlush >= minFlush &&
    minFlushEnabled >= 1 &&
    minFlushEnabled <= maxFlush;
  if (!httpOk || !flushOk) {
    return false;
  }
  const timeoutSec = cfg.httpTimeoutMs / 1000;
  if (timeoutSec < minHttp || timeoutSec > maxHttp) {
    if (chrome.IDE_HTTP_TIMEOUT_INVALID) {
      log(chrome.IDE_HTTP_TIMEOUT_INVALID);
    }
    return false;
  }
  if (
    cfg.autoFlushSeconds < minFlush ||
    cfg.autoFlushSeconds > maxFlush ||
    (cfg.autoFlushSeconds > 0 && cfg.autoFlushSeconds < minFlushEnabled)
  ) {
    if (chrome.IDE_AUTO_FLUSH_INVALID) {
      log(chrome.IDE_AUTO_FLUSH_INVALID);
    }
    return false;
  }
  return true;
}

/** Accept only http(s) absolute bases; no invented host. */
function parseApiBase(raw: string): string {
  const parsed = parseApiBasePure(raw);
  if (!parsed && raw.trim()) {
    if (chrome.IDE_CONFIG_API_URL_INVALID) {
      log(chrome.IDE_CONFIG_API_URL_INVALID);
    }
  }
  return parsed;
}

function clientVersionHeader(): string {
  return formatClientVersion(extensionVersion);
}

function userAgentHeader(): string {
  return formatUserAgent(extensionVersion);
}

async function apiFetch(
  base: string,
  path: string,
  init: RequestInit,
  timeoutMs: number,
): Promise<Response> {
  if (!base) {
    throw new Error("api base empty");
  }
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    const headers = new Headers(init.headers);
    if (!headers.has("Accept")) {
      headers.set("Accept", "application/json");
    }
    if (!headers.has("User-Agent")) {
      headers.set("User-Agent", userAgentHeader());
    }
    if (!headers.has("X-Client-Version")) {
      headers.set("X-Client-Version", clientVersionHeader());
    }
    if (!headers.has("X-Trim-Agent-Id")) {
      headers.set("X-Trim-Agent-Id", "ide");
    }
    if (!headers.has("X-Hardware-UUID")) {
      headers.set("X-Hardware-UUID", ideHardwareUUID());
    }
    const workspaceId = (
      vscode.workspace.getConfiguration("trim").get<string>("workspaceId") || ""
    ).trim();
    if (workspaceId && !headers.has("X-Workspace-Id")) {
      headers.set("X-Workspace-Id", workspaceId);
    }
    // Never follow redirects with Authorization - avoids Bearer leak on open redirect.
    return await fetch(joinApi(base, path), {
      ...init,
      headers,
      signal: ctrl.signal,
      redirect: "error",
    });
  } finally {
    clearTimeout(timer);
  }
}

function chromeComplete(messages: IdeChrome): boolean {
  return REQUIRED_CHROME.every((code) => {
    const v = messages[code];
    return typeof v === "string" && v.trim().length > 0;
  });
}

type QuotaExhaustedPayload = {
  remaining?: number;
  unlimited?: boolean;
  code?: string;
  tier_upgrade?: string;
  exhausted_title?: string;
  exhausted_body?: string;
  upgrade_action_label?: string;
  error?: string;
};

async function offerQuotaUpgrade(payload: QuotaExhaustedPayload): Promise<void> {
  if (quotaExhaustedNotified) {
    return;
  }
  if (payload.unlimited === true) {
    return;
  }
  const code = (payload.code || "").trim();
  const remaining =
    typeof payload.remaining === "number" && Number.isFinite(payload.remaining)
      ? payload.remaining
      : undefined;
  const exhausted =
    remaining !== undefined
      ? remaining <= 0
      : code === "QUOTA_EXHAUSTED" || code === "WORKSPACE_QUOTA_EXHAUSTED";
  if (!exhausted) {
    return;
  }
  const title =
    (payload.exhausted_title || "").trim() || chrome.IDE_QUOTA_EXHAUSTED_TITLE?.trim() || "";
  const body =
    (payload.exhausted_body || "").trim() || chrome.IDE_QUOTA_EXHAUSTED_BODY?.trim() || "";
  const action =
    (payload.upgrade_action_label || "").trim() || chrome.IDE_QUOTA_UPGRADE_ACTION?.trim() || "";
  const upgradeURL = (payload.tier_upgrade || "").trim();
  const msg =
    [title, body].filter(Boolean).join(" - ") ||
    (typeof payload.error === "string" ? payload.error.trim() : "");
  if (!msg) {
    return;
  }
  quotaExhaustedNotified = true;
  if (action && upgradeURL) {
    const picked = await vscode.window.showWarningMessage(msg, action);
    if (picked === action) {
      await vscode.env.openExternal(vscode.Uri.parse(upgradeURL));
    }
    return;
  }
  void vscode.window.showWarningMessage(msg);
}

/** Same conversion path as the web dashboard banner: remaining <= 0 → Upgrade via tier_upgrade. */
async function checkQuotaExhausted(
  context: vscode.ExtensionContext,
  cfg: TrimConfig,
): Promise<void> {
  if (quotaExhaustedNotified || !cfg.apiBase) {
    return;
  }
  const apiKey = (await context.secrets.get(SECRET_KEY))?.trim();
  if (!apiKey) {
    return;
  }
  try {
    const res = await apiFetch(
      cfg.apiBase,
      "/api/v1/me/quota",
      {
        method: "GET",
        headers: { Authorization: `Bearer ${apiKey}` },
      },
      cfg.httpTimeoutMs,
    );
    if (!res.ok) {
      log(`quota check HTTP ${res.status}`);
      return;
    }
    const data = (await res.json()) as QuotaExhaustedPayload;
    await offerQuotaUpgrade(data);
  } catch (err) {
    log(`quota check failed: ${err instanceof Error ? err.message : String(err)}`);
  }
}

async function resolveAutoStartPreference(
  context: vscode.ExtensionContext,
  cfg: TrimConfig,
): Promise<"on" | "off" | "unset"> {
  if (autostartManagedOff()) {
    return "off";
  }
  const local = vscode.workspace.getConfiguration("trim").get<string>("autoStartWithIde");
  if (local === "off") {
    return "off";
  }
  if (local === "on") {
    return "on";
  }
  const key = await context.secrets.get(SECRET_KEY);
  if (key?.trim() && cfg.apiBase) {
    try {
      const res = await apiFetch(
        cfg.apiBase,
        "/api/v1/me/preferences",
        {
          method: "GET",
          headers: { Authorization: `Bearer ${key.trim()}` },
        },
        cfg.httpTimeoutMs,
      );
      if (res.ok) {
        const data = (await res.json()) as { auto_start_with_ide?: unknown };
        if (typeof data.auto_start_with_ide === "boolean") {
          return data.auto_start_with_ide ? "on" : "off";
        }
      }
    } catch (err) {
      log(`preferences fetch failed: ${err instanceof Error ? err.message : String(err)}`);
    }
  }
  const def = (chrome.DEFAULT_AUTO_START_WITH_IDE || "").trim().toLowerCase();
  if (def === "true") {
    return "on";
  }
  if (def === "false") {
    return "off";
  }
  return "unset";
}

/** DO_NOT_TRACK / TRIM_AUTOSTART_DISABLED / ~/.config/trim/autostart.off - managed-off. */
function autostartManagedOff(): boolean {
  const env = process.env;
  if (env.DO_NOT_TRACK === "1") {
    return true;
  }
  if (env.TRIM_AUTOSTART_DISABLED === "1") {
    return true;
  }
  try {
    const marker = join(homedir(), ".config", "trim", "autostart.off");
    if (existsSync(marker)) {
      return true;
    }
  } catch {
    // if home unreadable, do not invent managed-off
  }
  return false;
}

/** Stop local proxy via DB chrome shutdown URL (used when pref/managed off). */
async function stopProxySoft(): Promise<void> {
  const url = (chrome.IDE_PROXY_SHUTDOWN_URL || "").trim();
  if (!url) {
    return;
  }
  const raw = (chrome.IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS || "").trim();
  const ms = Number.parseInt(raw, 10);
  const minMs = Number.parseInt((chrome.AUTOSTART_TIMEOUT_MIN_MS || "").trim(), 10);
  const maxMs = Number.parseInt((chrome.AUTOSTART_TIMEOUT_MAX_MS || "").trim(), 10);
  const boundsOk =
    Number.isFinite(minMs) && Number.isFinite(maxMs) && minMs >= 1 && maxMs >= 1 && minMs <= maxMs;
  if (!boundsOk || !Number.isFinite(ms) || ms < minMs || ms > maxMs) {
    log("IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS / bounds invalid (fail-closed)");
    return;
  }
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), ms);
  try {
    await fetch(url, { method: "POST", signal: ctrl.signal, redirect: "error" });
  } catch (err) {
    log(`proxy stop soft: ${err instanceof Error ? err.message : String(err)}`);
  } finally {
    clearTimeout(timer);
  }
}

async function requestProxyShutdown(): Promise<void> {
  const stopOnQuit = (chrome.IDE_AUTOSTART_STOP_ON_QUIT || "").trim().toLowerCase();
  if (stopOnQuit !== "true") {
    return;
  }
  await stopProxySoft();
}

async function proxyHealthy(healthUrl: string, timeoutMs: number): Promise<boolean> {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    const res = await fetch(healthUrl, { method: "GET", signal: ctrl.signal, redirect: "error" });
    if (!res.ok) {
      return false;
    }
    const body = (await res.json()) as { status?: string; proxy?: string };
    return body.status === "ok" && body.proxy === "trim";
  } catch {
    return false;
  } finally {
    clearTimeout(timer);
  }
}

function spawnTrimStart(): Promise<void> {
  return new Promise((resolve, reject) => {
    // Everyday IDE path: same enforcer bit as `trim daemon run` so dashboard
    // uncheck stops this process. Manual CLI `trim start` does not set it.
    const env = { ...process.env, TRIM_AUTOSTART_ENFORCER: "1" };
    const child = spawn("trim", ["start"], {
      detached: true,
      stdio: "ignore",
      shell: process.platform === "win32",
      windowsHide: true,
      env,
    });
    child.on("error", (err) => {
      reject(err);
    });
    child.unref();
    resolve();
  });
}

function markProxyStatus(status: vscode.StatusBarItem | undefined, ok: boolean): void {
  if (!status) return;
  if (ok && chrome.IDE_AUTOSTART_STATUS_OK) {
    status.text = chrome.IDE_AUTOSTART_STATUS_OK;
  } else if (!ok && chrome.IDE_AUTOSTART_STATUS_FAILED) {
    status.text = chrome.IDE_AUTOSTART_STATUS_FAILED;
  }
}

function warnProxyFailedSoft(): void {
  if (chrome.IDE_AUTOSTART_WARN_FAILED) {
    void vscode.window.showWarningMessage(chrome.IDE_AUTOSTART_WARN_FAILED);
  }
}

async function ensureProxyAutostart(
  context: vscode.ExtensionContext,
  cfg: TrimConfig,
  status?: vscode.StatusBarItem,
): Promise<void> {
  if (autostartManagedOff()) {
    if (chrome.IDE_AUTOSTART_SKIPPED_MANAGED) {
      log(chrome.IDE_AUTOSTART_SKIPPED_MANAGED);
    }
    await stopProxySoft();
    return;
  }
  const local = vscode.workspace.getConfiguration("trim").get<string>("autoStartWithIde");
  if (local === "off") {
    if (chrome.IDE_AUTOSTART_SKIPPED_LOCAL_OFF) {
      log(chrome.IDE_AUTOSTART_SKIPPED_LOCAL_OFF);
    }
    await stopProxySoft();
    return;
  }
  const pref = await resolveAutoStartPreference(context, cfg);
  if (pref === "off") {
    if (chrome.IDE_AUTOSTART_SKIPPED_OFF) {
      log(chrome.IDE_AUTOSTART_SKIPPED_OFF);
    }
    // Dashboard/cloud off: stop running proxy on this IDE open (local enforcer).
    await stopProxySoft();
    return;
  }
  if (pref === "unset") {
    if (chrome.IDE_AUTOSTART_SKIPPED_UNSET) {
      log(chrome.IDE_AUTOSTART_SKIPPED_UNSET);
    }
    return;
  }
  const healthUrl = (chrome.IDE_PROXY_HEALTH_URL || "").trim();
  if (!healthUrl) {
    log("IDE_PROXY_HEALTH_URL empty (fail-closed)");
    return;
  }
  if (chrome.IDE_AUTOSTART_ENSURING) {
    log(chrome.IDE_AUTOSTART_ENSURING);
  }
  if (await proxyHealthy(healthUrl, cfg.httpTimeoutMs)) {
    if (chrome.IDE_AUTOSTART_PROXY_OK) {
      log(chrome.IDE_AUTOSTART_PROXY_OK);
    }
    markProxyStatus(status, true);
    return;
  }
  try {
    await spawnTrimStart();
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    if (await proxyHealthy(healthUrl, cfg.httpTimeoutMs)) {
      if (chrome.IDE_AUTOSTART_PROXY_OK) {
        log(chrome.IDE_AUTOSTART_PROXY_OK);
      }
      markProxyStatus(status, true);
      return;
    }
    if (/ENOENT|not found/i.test(msg)) {
      if (chrome.IDE_AUTOSTART_CLI_MISSING) {
        log(chrome.IDE_AUTOSTART_CLI_MISSING);
      }
    } else {
      const fmt = chrome.IDE_AUTOSTART_PROXY_FAILED_FMT;
      if (fmt) {
        log(fmt.includes("%s") ? fmt.replace("%s", msg) : `${fmt} ${msg}`);
      }
    }
    markProxyStatus(status, false);
    warnProxyFailedSoft();
    return;
  }
  const settleRaw = (chrome.IDE_AUTOSTART_SETTLE_MS || "").trim();
  const settleMs = Number.parseInt(settleRaw, 10);
  const minMs = Number.parseInt((chrome.AUTOSTART_TIMEOUT_MIN_MS || "").trim(), 10);
  const maxMs = Number.parseInt((chrome.AUTOSTART_TIMEOUT_MAX_MS || "").trim(), 10);
  const boundsOk =
    Number.isFinite(minMs) && Number.isFinite(maxMs) && minMs >= 1 && maxMs >= 1 && minMs <= maxMs;
  // settle 0 allowed (no wait); positive must be within DB bounds
  if (
    !boundsOk ||
    !Number.isFinite(settleMs) ||
    settleMs < 0 ||
    (settleMs > 0 && (settleMs < minMs || settleMs > maxMs))
  ) {
    log("IDE_AUTOSTART_SETTLE_MS / bounds invalid (fail-closed)");
    return;
  }
  await new Promise((r) => setTimeout(r, settleMs));
  if (await proxyHealthy(healthUrl, cfg.httpTimeoutMs)) {
    if (chrome.IDE_AUTOSTART_PROXY_STARTED) {
      log(chrome.IDE_AUTOSTART_PROXY_STARTED);
    }
    markProxyStatus(status, true);
    return;
  }
  const fmt = chrome.IDE_AUTOSTART_PROXY_FAILED_FMT;
  if (fmt) {
    const detail = (chrome.IDE_AUTOSTART_HEALTH_AFTER_START_FAILED || "").trim();
    if (detail) {
      log(fmt.includes("%s") ? fmt.replace("%s", detail) : fmt);
    } else {
      log(fmt.includes("%s") ? fmt.replace("%s", "") : fmt);
    }
  }
  markProxyStatus(status, false);
  warnProxyFailedSoft();
}

const IDE_CHROME_CACHE_KEY = "trim.ideChromeCache";

async function loadChrome(context: vscode.ExtensionContext, cfg: TrimConfig): Promise<boolean> {
  if (!cfg.apiBase) {
    if (chrome.IDE_CONFIG_API_URL_EMPTY) {
      log(chrome.IDE_CONFIG_API_URL_EMPTY);
    }
    return false;
  }
  try {
    const res = await apiFetch(
      cfg.apiBase,
      "/api/v1/public/ide-chrome",
      { method: "GET" },
      cfg.httpTimeoutMs,
    );
    if (!res.ok) {
      log(`ide-chrome HTTP ${res.status}`);
      return tryLoadCachedChrome(context);
    }
    const data = (await res.json()) as { messages?: IdeChrome };
    if (!data.messages || typeof data.messages !== "object") {
      log("ide-chrome response missing messages object");
      return tryLoadCachedChrome(context);
    }
    if (!chromeComplete(data.messages)) {
      log("ide-chrome missing required site_messages codes (fail-closed)");
      return tryLoadCachedChrome(context);
    }
    chrome = data.messages;
    await context.globalState.update(IDE_CHROME_CACHE_KEY, data.messages);
    log("ide-chrome loaded");
    return true;
  } catch (err) {
    log(`ide-chrome request failed: ${err instanceof Error ? err.message : String(err)}`);
    return tryLoadCachedChrome(context);
  }
}

/** Last-known-good ide-chrome from prior API sync (DB-sourced). Never invent. */
function tryLoadCachedChrome(context: vscode.ExtensionContext): boolean {
  const cached = context.globalState.get<IdeChrome>(IDE_CHROME_CACHE_KEY);
  if (!cached || typeof cached !== "object" || !chromeComplete(cached)) {
    return false;
  }
  chrome = cached;
  log("ide-chrome loaded from last-known-good cache");
  return true;
}

function refreshStatus(status: vscode.StatusBarItem, c: Counters): void {
  const pending = c.tabShown + c.tabAccepted + c.linesAdded + c.linesDeleted;
  if (pending > 0 && chrome.IDE_STATUS_PENDING_FMT) {
    status.text = chrome.IDE_STATUS_PENDING_FMT.replace("%d", String(pending));
    return;
  }
  status.text = chrome.IDE_STATUS_IDLE?.trim() || "";
}

async function flush(
  context: vscode.ExtensionContext,
  counters: Counters,
  cfg: TrimConfig,
): Promise<boolean> {
  if (flushInFlight) {
    return flushInFlight;
  }
  flushInFlight = doFlush(context, counters, cfg).finally(() => {
    flushInFlight = undefined;
  });
  return flushInFlight;
}

async function doFlush(
  context: vscode.ExtensionContext,
  counters: Counters,
  cfg: TrimConfig,
): Promise<boolean> {
  const pending =
    counters.tabShown + counters.tabAccepted + counters.linesAdded + counters.linesDeleted;
  if (pending === 0) {
    return false;
  }
  if (!cfg.apiBase) {
    log("flush skipped: apiUrl empty");
    return false;
  }
  if (
    !chrome.IDE_EVENT_MODE?.trim() ||
    !chrome.IDE_EVENT_STATUS?.trim() ||
    !chrome.IDE_EVENT_MODEL?.trim()
  ) {
    log("flush skipped: event chrome missing");
    return false;
  }

  const apiKey = await context.secrets.get(SECRET_KEY);
  if (!apiKey) {
    log("flush skipped: API key not set (Trim: Set API Key)");
    return false;
  }

  // Snapshot then clear only after success (at-least-once; avoids silent drop on failure).
  const snapshot = { ...counters };
  const body = {
    mode: chrome.IDE_EVENT_MODE,
    status: chrome.IDE_EVENT_STATUS,
    tokens_before: 0,
    tokens_after: 0,
    latency_ms: 0,
    model: chrome.IDE_EVENT_MODEL,
    tab_suggestions_shown: snapshot.tabShown,
    tab_suggestions_accepted: snapshot.tabAccepted,
    ai_lines_added: snapshot.linesAdded,
    ai_lines_deleted: snapshot.linesDeleted,
  };

  try {
    const attemptFlush = async (): Promise<Response> =>
      apiFetch(
        cfg.apiBase,
        "/api/v1/me/events",
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${apiKey}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(body),
        },
        cfg.httpTimeoutMs,
      );

    let res: Response;
    try {
      res = await attemptFlush();
    } catch (firstErr) {
      // One immediate retry on transient network failure (no invented backoff delay).
      log(
        `flush network error (retrying once): ${
          firstErr instanceof Error ? firstErr.message : String(firstErr)
        }`,
      );
      res = await attemptFlush();
    }
    if (!res.ok && res.status >= 500) {
      log(`flush HTTP ${res.status} (retrying once)`);
      res = await attemptFlush();
    }
    if (!res.ok) {
      let apiMsg = "";
      let upgradeURL = "";
      let exhaustedCode = "";
      let exhaustedTitle = "";
      let exhaustedBody = "";
      let upgradeLabel = "";
      try {
        const raw = await res.text();
        const parsed = JSON.parse(raw) as {
          error?: string;
          message?: string;
          code?: string;
          tier_upgrade?: string;
          exhausted_title?: string;
          exhausted_body?: string;
          upgrade_action_label?: string;
        };
        apiMsg =
          (typeof parsed.error === "string" && parsed.error.trim()) ||
          (typeof parsed.message === "string" && parsed.message.trim()) ||
          "";
        exhaustedCode = typeof parsed.code === "string" ? parsed.code.trim() : "";
        upgradeURL = typeof parsed.tier_upgrade === "string" ? parsed.tier_upgrade.trim() : "";
        exhaustedTitle =
          typeof parsed.exhausted_title === "string" ? parsed.exhausted_title.trim() : "";
        exhaustedBody =
          typeof parsed.exhausted_body === "string" ? parsed.exhausted_body.trim() : "";
        upgradeLabel =
          typeof parsed.upgrade_action_label === "string" ? parsed.upgrade_action_label.trim() : "";
      } catch {
        apiMsg = "";
      }
      log(`flush HTTP ${res.status}${apiMsg ? `: ${apiMsg}` : ""}`);
      const quotaExhausted =
        res.status === 402 &&
        (exhaustedCode === "QUOTA_EXHAUSTED" || exhaustedCode === "WORKSPACE_QUOTA_EXHAUSTED");
      if (quotaExhausted) {
        await offerQuotaUpgrade({
          code: exhaustedCode,
          tier_upgrade: upgradeURL,
          exhausted_title: exhaustedTitle,
          exhausted_body: exhaustedBody,
          upgrade_action_label: upgradeLabel,
          error: apiMsg,
        });
        return false;
      }
      if (chrome.IDE_TELEMETRY_FAILED_FMT) {
        void vscode.window.showWarningMessage(
          chrome.IDE_TELEMETRY_FAILED_FMT.replace("%d", String(res.status)).replace("%s", apiMsg),
        );
      }
      return false;
    }
    counters.tabShown = Math.max(0, counters.tabShown - snapshot.tabShown);
    counters.tabAccepted = Math.max(0, counters.tabAccepted - snapshot.tabAccepted);
    counters.linesAdded = Math.max(0, counters.linesAdded - snapshot.linesAdded);
    counters.linesDeleted = Math.max(0, counters.linesDeleted - snapshot.linesDeleted);
    await persistCounters(context, counters);
    log(`flush ok (pending was ${pending})`);
    void checkQuotaExhausted(context, cfg);
    return true;
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    log(`flush network error: ${msg}`);
    if (chrome.IDE_TELEMETRY_NETWORK_FMT) {
      void vscode.window.showWarningMessage(chrome.IDE_TELEMETRY_NETWORK_FMT.replace("%s", msg));
    }
    return false;
  }
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  extensionVersion =
    (context.extension.packageJSON as { version?: string }).version?.trim() || "0.0.0";

  output = vscode.window.createOutputChannel("Trim");
  context.subscriptions.push(output);
  log(`activate trim-ide/${extensionVersion}`);

  // Prefer last-known-good DB chrome before config error copy (no invent English).
  tryLoadCachedChrome(context);

  const cfg = readConfig();
  if (!cfg) {
    return;
  }

  const ready = await loadChrome(context, cfg);
  if (!ready) {
    // Fail closed: no invented status-bar / toast chrome when API messages unavailable.
    // Operators: open View → Output → Trim for the reason.
    return;
  }
  if (!configWithinChromeBounds(cfg)) {
    return;
  }

  const counters = loadPersistedCounters(context);
  runtime = { context, counters };
  if (counters.tabShown + counters.tabAccepted + counters.linesAdded + counters.linesDeleted > 0) {
    log("restored pending telemetry counters from globalState");
  }

  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  status.command = "trim.flushTelemetry";
  status.tooltip = chrome.IDE_STATUS_TOOLTIP;
  context.subscriptions.push(status);
  refreshStatus(status, counters);

  // Fail soft: never brick IDE if proxy start fails.
  await ensureProxyAutostart(context, cfg, status);

  // Match web dashboard: when remaining <= 0, surface Upgrade (DB chrome + tier_upgrade URL).
  void checkQuotaExhausted(context, cfg);

  const liveConfig = (): TrimConfig => {
    const next = readConfig();
    return next ?? cfg;
  };

  // While IDE stays open: re-check cloud/local pref (dashboard uncheck → stop).
  // Interval + bounds from site_messages (0 = disabled). No invent min/max.
  {
    const minMs = Number.parseInt((chrome.AUTOSTART_PREF_POLL_MIN_MS || "").trim(), 10);
    const maxMs = Number.parseInt((chrome.AUTOSTART_PREF_POLL_MAX_MS || "").trim(), 10);
    const pollRaw = (chrome.IDE_AUTOSTART_PREF_POLL_MS || "").trim();
    const pollMs = Number.parseInt(pollRaw, 10);
    const boundsOk =
      Number.isFinite(minMs) &&
      Number.isFinite(maxMs) &&
      minMs >= 1 &&
      maxMs >= 1 &&
      minMs <= maxMs;
    if (
      !boundsOk ||
      !Number.isFinite(pollMs) ||
      pollMs < 0 ||
      (pollMs > 0 && (pollMs < minMs || pollMs > maxMs))
    ) {
      log("IDE_AUTOSTART_PREF_POLL_MS / bounds invalid (fail-closed; poll disabled)");
    } else if (pollMs > 0) {
      const pollTimer = setInterval(() => {
        void ensureProxyAutostart(context, liveConfig(), status);
      }, pollMs);
      context.subscriptions.push({ dispose: () => clearInterval(pollTimer) });
      log(`autostart pref poll every ${pollMs}ms`);
    }
  }

  context.subscriptions.push(
    vscode.commands.registerCommand("trim.setApiKey", async () => {
      const key = await vscode.window.showInputBox({
        title: chrome.IDE_API_KEY_TITLE,
        prompt: chrome.IDE_API_KEY_PROMPT,
        password: true,
        ignoreFocusOut: true,
      });
      if (!key?.trim()) {
        return;
      }
      await context.secrets.store(SECRET_KEY, key.trim());
      if (chrome.IDE_API_KEY_SAVED) {
        void vscode.window.showInformationMessage(chrome.IDE_API_KEY_SAVED);
      }
      log("API key stored in SecretStorage");
      quotaExhaustedNotified = false;
      void checkQuotaExhausted(context, liveConfig());
    }),
    vscode.commands.registerCommand("trim.clearApiKey", async () => {
      await context.secrets.delete(SECRET_KEY);
      await context.globalState.update(COUNTERS_STATE_KEY, undefined);
      await context.globalState.update(IDE_CHROME_CACHE_KEY, undefined);
      counters.tabShown = 0;
      counters.tabAccepted = 0;
      counters.linesAdded = 0;
      counters.linesDeleted = 0;
      refreshStatus(status, counters);
      if (chrome.IDE_API_KEY_CLEARED) {
        void vscode.window.showInformationMessage(chrome.IDE_API_KEY_CLEARED);
      }
      log("API key and local extension state cleared");
    }),
    vscode.commands.registerCommand("trim.copyHardwareId", async () => {
      const hw = ideHardwareUUID();
      await vscode.env.clipboard.writeText(hw);
      if (chrome.IDE_HARDWARE_ID_COPIED) {
        void vscode.window.showInformationMessage(chrome.IDE_HARDWARE_ID_COPIED);
      }
      log(`hardware id copied (${hw.slice(0, 12)}…)`);
    }),
    vscode.commands.registerCommand("trim.markTabShown", () => {
      counters.tabShown += 1;
      refreshStatus(status, counters);
      void persistCounters(context, counters);
    }),
    vscode.commands.registerCommand("trim.markTabAccepted", () => {
      counters.tabShown += 1;
      counters.tabAccepted += 1;
      refreshStatus(status, counters);
      void persistCounters(context, counters);
    }),
    vscode.commands.registerCommand("trim.flushTelemetry", async () => {
      const ok = await flush(context, counters, liveConfig());
      if (ok && chrome.IDE_TELEMETRY_FLUSHED) {
        void vscode.window.showInformationMessage(chrome.IDE_TELEMETRY_FLUSHED);
      }
      refreshStatus(status, counters);
    }),
    vscode.commands.registerCommand("trim._onInlineSuggestCommit", (...args: unknown[]) => {
      counters.tabShown += 1;
      counters.tabAccepted += 1;
      refreshStatus(status, counters);
      void persistCounters(context, counters);
      return vscode.commands.executeCommand("editor.action.inlineSuggest.commit", ...args);
    }),
  );

  if (cfg.trackDocumentEdits === true) {
    context.subscriptions.push(
      vscode.workspace.onDidChangeTextDocument((e) => {
        if (e.document.uri.scheme !== "file" && e.document.uri.scheme !== "untitled") {
          return;
        }
        const minLines = liveConfig().minLinesForAiHeuristic;
        if (minLines < 1) {
          return;
        }
        let changed = false;
        for (const change of e.contentChanges) {
          const added = Math.max(0, change.text.split(/\r?\n/).length - 1);
          const removed = change.range.end.line - change.range.start.line;
          if (added >= minLines) {
            counters.linesAdded += added;
            changed = true;
          }
          if (removed >= minLines) {
            counters.linesDeleted += removed;
            changed = true;
          }
        }
        if (changed) {
          refreshStatus(status, counters);
          void persistCounters(context, counters);
        }
      }),
    );
  }

  // Auto-flush: only when settings already passed configWithinChromeBounds.
  // 0 = off. Enabled floor/ceiling = IDE_AUTO_FLUSH_* from site_messages (no invent 15/600).
  if (cfg.autoFlushSeconds > 0) {
    const seconds = cfg.autoFlushSeconds;
    const timer = setInterval(() => {
      void flush(context, counters, liveConfig()).then(() => refreshStatus(status, counters));
    }, seconds * 1000);
    context.subscriptions.push({ dispose: () => clearInterval(timer) });
    log(`auto-flush every ${seconds}s`);
  }

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("trim.autoStartWithIde")) {
        // Local override changed → re-enforce without requiring window reload.
        void ensureProxyAutostart(context, liveConfig(), status);
        return;
      }
      if (!e.affectsConfiguration("trim")) {
        return;
      }
      log("trim.* settings changed - reload window to re-bind chrome/timers");
    }),
  );

  status.show();
  log("ready");
}

export function deactivate(): Thenable<void> | undefined {
  const rt = runtime;
  const cfg = readConfig();
  runtime = undefined;

  const finish = (): void => {
    flushInFlight = undefined;
    chrome = {};
    log("deactivate");
  };

  const afterShutdown = (): Thenable<void> | undefined => {
    if (!rt) {
      finish();
      return undefined;
    }
    // Persist first so a quit race cannot drop pending counters.
    return persistCounters(rt.context, rt.counters).then(() => {
      if (cfg?.apiBase && chrome.IDE_EVENT_MODE?.trim()) {
        return flush(rt.context, rt.counters, cfg).then(finish, finish);
      }
      finish();
      return undefined;
    }, finish);
  };

  // Explicit quit policy from site_messages IDE_AUTOSTART_STOP_ON_QUIT (default leave daemon).
  return requestProxyShutdown().then(afterShutdown, afterShutdown);
}
