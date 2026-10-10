import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  clientVersionHeader,
  emptyCounters,
  joinApi,
  normalizeCounters,
  parseApiBase,
  pendingTotal,
  userAgentHeader,
} from "../util";

describe("parseApiBase", () => {
  it("accepts https origin", () => {
    assert.equal(parseApiBase("https://api.use-trim.com"), "https://api.use-trim.com");
  });
  it("strips trailing slash", () => {
    assert.equal(parseApiBase("https://api.use-trim.com/"), "https://api.use-trim.com");
  });
  it("keeps API subpath", () => {
    assert.equal(parseApiBase("https://example.com/trim/"), "https://example.com/trim");
  });
  it("rejects non-http(s)", () => {
    assert.equal(parseApiBase("ftp://evil.example"), "");
    assert.equal(parseApiBase("javascript:alert(1)"), "");
  });
  it("rejects empty / garbage", () => {
    assert.equal(parseApiBase(""), "");
    assert.equal(parseApiBase("not a url"), "");
  });
});

describe("joinApi", () => {
  it("joins relative path", () => {
    assert.equal(
      joinApi("https://api.use-trim.com", "/api/v1/public/ide-chrome"),
      "https://api.use-trim.com/api/v1/public/ide-chrome",
    );
  });
  it("handles base with trailing slash", () => {
    assert.equal(
      joinApi("https://api.use-trim.com/", "api/v1/me/events"),
      "https://api.use-trim.com/api/v1/me/events",
    );
  });
});

describe("counters", () => {
  it("emptyCounters is zeroed", () => {
    assert.deepEqual(emptyCounters(), {
      tabShown: 0,
      tabAccepted: 0,
      linesAdded: 0,
      linesDeleted: 0,
    });
  });
  it("normalizeCounters floors and drops invalid", () => {
    assert.deepEqual(
      normalizeCounters({
        tabShown: 2.9,
        tabAccepted: -1,
        linesAdded: Number.NaN,
        linesDeleted: 4,
      }),
      { tabShown: 2, tabAccepted: 0, linesAdded: 0, linesDeleted: 4 },
    );
  });
  it("pendingTotal sums", () => {
    assert.equal(pendingTotal({ tabShown: 1, tabAccepted: 2, linesAdded: 3, linesDeleted: 4 }), 10);
  });
});

describe("headers", () => {
  it("formats client + UA", () => {
    assert.equal(clientVersionHeader("1.0.0"), "trim-ide/1.0.0");
    assert.equal(userAgentHeader("1.0.0"), "TrimIDE/1.0.0");
  });
});
