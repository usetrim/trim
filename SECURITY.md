# Security Policy

## Reporting a Vulnerability

Please report security issues privately to security@use-trim.com (or the address listed in your fork's SECURITY contacts).

Do not open public GitHub issues for vulnerabilities.

Include:

- Affected component (`cli`, `server`, `apps/web`)
- Version or commit hash
- Steps to reproduce
- Impact assessment

We aim to acknowledge reports within 72 hours and ship fixes as quickly as possible.

For contributor legal acknowledgments, see [CLA.md](./CLA.md) and [CONTRIBUTING.md](./CONTRIBUTING.md).


## Supported Versions

| Version | Supported |
| ------- | --------- |
| latest  | Yes       |
| older   | Best effort |

## Scope

In scope: authentication bypass, quota bypass, secret leaks, remote code execution, SSRF against the proxy.

Out of scope: denial of service against third-party LLM providers, social engineering, physical access attacks.

## Defense layers (ops checklist)

Trim enforces quotas and identity on the server only. Clearing local CLI state does not reset credits. `plan_catalog.unlimited` is an operator growth dial (default off): when enabled for a plan tier, AuthQuota skips debit for that tier only. Client apps cannot invent Unlimited; reports of “quota bypass” that require flipping a catalog flag are out of scope as product configuration, not vulnerabilities.

| Layer | What it covers | Where / how to enable |
| ----- | -------------- | --------------------- |
| 1 Network / VPN | Cloudflare ASN headers, optional GeoLite MMDB or CIDR blocklists, ipwho.is fallback | `TRIM_GEOLITE_*_MMDB_PATH`, CF proxy |
| 2 Transport fingerprint | JA3 / JA4 account graph (edge-forwarded) | `CF-JA4` / `X-JA4` headers |
| 3 Hardware identity | Device UUID bound to API keys (`api_keys.hardware_uuid` enforced on use) | CLI `X-Hardware-UUID` |
| 4 Request integrity | Timestamp window + CLI HMAC | `TRIM_CLI_HMAC_SECRET` |
| 5 Social identity | Google / GitHub / GitLab OAuth only (DB allow-list) | Supabase providers + `ALLOWED_AUTH_PROVIDERS` |
| 6 Server quotas | Redis + Postgres credit ledger; `plan_catalog.unlimited` skips debit when operator enables it | Always on for cloud mode; dial via Admin → Plans |
| 7 Multi-account graph | Hardware + JA4 fan-out limits | `TRIM_MAX_ACCOUNTS_PER_*` |
| 8 Canary / deception | Decoy paths and canary keys | Built into gateway |

Additional hot-path controls (same middleware stack):

| Control | What it covers | Where / how |
| ------- | -------------- | ----------- |
| Postgres IP/ASN denylist | Admin `ip_denylist` / `asn_denylist` on all `/api/v1` + Protect (auth/proxy/decoy) | Redis-cached; cloud fails closed on DB errors; ASN from `CF-ASN` or GeoLite |
| Agent identity | `X-Trim-Agent-Id` (`cli` / `ide` / `ci`) required for API keys | `agent_identity_catalog`; optional per-key lock |
| API key devices | Every API key must match allowlisted `X-Hardware-UUID` (default max 2); unbound keys rejected | `api_key_devices` + JWT register; no stolen-key auto-enroll |

Optional offline GeoLite: download MaxMind GeoLite2-Anonymous-IP / ASN `.mmdb` files and set:

```
TRIM_GEOLITE_ANONYMOUS_MMDB_PATH=/etc/trim/GeoLite2-Anonymous-IP.mmdb
TRIM_GEOLITE_ASN_MMDB_PATH=/etc/trim/GeoLite2-ASN.mmdb
```

Plain CIDR/ASN text files remain supported for air-gapped exports.
