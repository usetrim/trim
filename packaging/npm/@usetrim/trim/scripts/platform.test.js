"use strict";

const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const { resolvePlatform } = require("./platform");

describe("resolvePlatform", () => {
  it("maps darwin arm64", () => {
    const p = resolvePlatform("darwin", "arm64");
    assert.equal(p.asset, "trim_darwin_arm64.tar.gz");
    assert.equal(p.binaryName, "trim");
  });

  it("maps linux x64", () => {
    const p = resolvePlatform("linux", "x64");
    assert.equal(p.asset, "trim_linux_amd64.tar.gz");
  });

  it("maps win32 x64 to zip", () => {
    const p = resolvePlatform("win32", "x64");
    assert.equal(p.asset, "trim_windows_amd64.zip");
    assert.equal(p.binaryName, "trim.exe");
  });

  it("rejects exotic arch", () => {
    assert.throws(() => resolvePlatform("linux", "ia32"), /Unsupported architecture/);
  });
});
