# Security Policy - Trim IDE

## Supported versions

| Version | Supported |
| --- | --- |
| 1.x | Yes |
| 0.x | Best-effort (upgrade to latest Marketplace / VSIX release) |

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security-sensitive reports.

Email: **security@use-trim.com** (or `support@use-trim.com` with subject `SECURITY` if security@ is unavailable)

Include:

- Extension version (`Trim IDE` in Extensions view)
- Editor (VS Code / Cursor) and OS
- Steps to reproduce / impact
- Whether an API key or user data could be exposed

We aim to acknowledge within **3 business days**.

## Hardening already in place

- API keys use editor **Secret Storage** (not workspace settings)
- HTTP(S) only for `trim.apiUrl`; `redirect: "error"` on fetches (no Bearer leak on redirects)
- Request timeouts; UI labels come from Trim Cloud (extension stays inactive until sync succeeds)
- Spawns `trim start` only (never Deep Mode compress on IDE open)

## Scope

This policy covers the **Trim IDE** extension sources under `extensions/trim-ide`. Hosted API / account security is covered by Trim Cloud operations and [https://use-trim.com/privacy](https://use-trim.com/privacy).
