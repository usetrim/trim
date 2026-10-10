#!/usr/bin/env node
/**
 * Fail CI/local publish if extensions/trim-ide/package.nls.json drifts from
 * IDE_CMD_* / IDE_CONFIG_* builtin strings in
 * server/internal/subscriptions/service.go (site_messages seed source).
 *
 * Usage: node scripts/check-ide-nls-sync.mjs
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const nlsPath = path.join(root, "extensions", "trim-ide", "package.nls.json");
const servicePath = path.join(
  root,
  "server",
  "internal",
  "subscriptions",
  "service.go"
);

const nls = JSON.parse(fs.readFileSync(nlsPath, "utf8"));
const service = fs.readFileSync(servicePath, "utf8");

const map = {
  "trim.command.setApiKey.title": "IDE_CMD_SET_API_KEY",
  "trim.command.clearApiKey.title": "IDE_CMD_CLEAR_API_KEY",
  "trim.command.markTabShown.title": "IDE_CMD_MARK_TAB_SHOWN",
  "trim.command.markTabAccepted.title": "IDE_CMD_MARK_TAB_ACCEPTED",
  "trim.command.flushTelemetry.title": "IDE_CMD_FLUSH_TELEMETRY",
  "trim.config.title": "IDE_CONFIG_TITLE",
  "trim.config.apiUrl": "IDE_CONFIG_API_URL",
  "trim.config.autoFlushSeconds": "IDE_CONFIG_AUTO_FLUSH",
  "trim.config.trackDocumentEdits": "IDE_CONFIG_TRACK_EDITS",
  "trim.config.minLines": "IDE_CONFIG_MIN_LINES",
};

function builtinFor(code) {
  const re = new RegExp(
    `case "${code}":\\s*\\n\\s*return "([^"]*)"`,
    "m"
  );
  const m = service.match(re);
  return m ? m[1] : null;
}

let failed = false;
for (const [nlsKey, code] of Object.entries(map)) {
  const expected = builtinFor(code);
  const actual = nls[nlsKey];
  if (expected == null) {
    console.error(`missing builtin MessageForCode(${code}) in service.go`);
    failed = true;
    continue;
  }
  if (actual !== expected) {
    console.error(
      `nls drift: ${nlsKey}\n  package.nls.json: ${JSON.stringify(actual)}\n  ${code}:          ${JSON.stringify(expected)}`
    );
    failed = true;
  }
}

if (failed) {
  process.exit(1);
}
console.log("ide package.nls.json matches IDE_CMD_* / IDE_CONFIG_* builtins");
