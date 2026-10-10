# Publish Trim IDE (operators)

Extension id: **`usetrim.trim-ide`**

| Channel | Status path |
| --- | --- |
| **VS Marketplace** | Live as `usetrim.trim-ide` - manual `.vsix` upload and/or optional `VSCE_PAT` CI |
| **Open VSX** (Cursor) | Live `1.0.0` - bootstrap PAT done; claim verified namespace → **Trusted Publishing (OIDC)** (no long-lived `OVSX_PAT`) |

CI: [`.github/workflows/publish-trim-ide.yml`](../../.github/workflows/publish-trim-ide.yml)

References (official):

- [Trusted Publishing on Open VSX](https://blogs.eclipse.org/post/tamas-cservenak/trusted-publishing-open-vsx) (Eclipse Foundation, Sep 2026)
- [Publishing Extensions](https://github.com/eclipse-openvsx/openvsx/wiki/Publishing-Extensions)
- [Managing Namespaces](https://github.com/EclipseFdn/open-vsx.org/wiki/Managing-Namespaces)
- [ovsx CLI - Trusted Publishing](https://www.npmjs.com/package/ovsx) (`ovsx` ≥ **1.1**; we use **^1.2**)

---

## Prerequisites (one-time)

1. **Publisher Agreement** signed on https://open-vsx.org/user-settings/profile  
   (Eclipse Foundation account + GitHub username linked - same GitHub user as Open VSX login).
2. **VS Marketplace** publisher id **`usetrim`** (already created if you followed the Marketplace path).
3. GitHub account age **≥ 1 year** to claim Open VSX namespace ownership (Eclipse policy).

---

## Phase A - Bootstrap Open VSX (PAT, once)

Trusted Publishing **cannot** publish the first version. You need a short-lived PAT for:

1. `create-namespace`
2. First `publish`

### A1) Create token

https://open-vsx.org/user-settings/tokens → **Generate New Token**  
Description: `trim-ide-bootstrap` → copy once.

### A2) Create namespace + publish

```powershell
# From the folder that contains trim-ide-1.0.0.vsix (Actions artifact), OR from extensions/trim-ide after npm run package
npx ovsx create-namespace usetrim -p YOUR_TOKEN
npx ovsx publish .\trim-ide-1.0.0.vsix -p YOUR_TOKEN
```

Confirm: https://open-vsx.org/extension/usetrim/trim-ide

Optional bootstrap via CI: add repo secret `OVSX_PAT`, then Actions → **Trim IDE extension** → Run workflow → `publish=true`.  
(`OVSX_PAT` wins over OIDC until you delete it.)

---

## Phase B - Claim verified namespace ownership (required for Trusted Publishing)

Creating a namespace only makes you a **contributor**. Trusted Publishing requires **owner** on a **verified** namespace.

1. Open: https://github.com/EclipseFdn/open-vsx.org/issues/new/choose  
2. Choose the **namespace ownership / claim** template.  
3. Request ownership of namespace **`usetrim`**.  
4. Because `usetrim` is also a **VS Marketplace** publisher, include proof you control it (template options), e.g.:
   - Marketplace publisher profile lists GitHub org `usetrim`, and you control that org, or  
   - Published Marketplace extension repo `https://github.com/usetrim/trim` + a commit by your GitHub user, or  
   - Temporary reader access for an Open VSX admin (remove after approval).
5. Wait until the issue is resolved and https://open-vsx.org shows the namespace / extension as **verified**.

Wiki: https://github.com/EclipseFdn/open-vsx.org/wiki/Managing-Namespaces

---

## Phase C - GitHub Environment + Trusted Publisher (permanent)

Eclipse guidance: Trusted Publisher registrations are **not branch-scoped**. Pin a deployment environment so only approved runs can publish.

### C1) Create GitHub Environment

1. https://github.com/usetrim/trim/settings/environments  
2. **New environment** → name exactly: **`open-vsx`**  
3. Optional (recommended): add required reviewers / wait timer for protection  
4. Save

### C2) Register on Open VSX

1. Open: https://open-vsx.org/user-settings/trusted-publishers  
   (or Access Tokens → **Set up a trusted publisher**)  
2. Namespace: **`usetrim`**  
3. GitHub Actions - exact values:

| Field | Value |
| --- | --- |
| Organization or user | `usetrim` |
| Repository | `trim` |
| Workflow filename | `publish-trim-ide.yml` |
| Environment | **`open-vsx`** (must match the GitHub Environment) |

4. Save. Registrations are immutable (delete + recreate to change). Matching uses GitHub numeric owner/repo IDs; workflow filename is matched by name.

---

## Phase D - Merge workflow + harden

1. Merge the branch that contains `permissions.id-token: write`, `environment: open-vsx`, and `ovsx publish --no-dependencies --trusted-publishing`.  
2. **Delete** GitHub secret `OVSX_PAT` (if set).  
3. **Revoke** the Open VSX bootstrap token.  
4. Actions → **Trim IDE extension** → Run workflow → `publish=true`  
   **or** publish a GitHub Release.  
5. Confirm Open VSX version shows **trusted publisher** on the listing.

Do **not** leave `OVSX_PAT` in the environment: ovsx always prefers PAT over OIDC.

---

## VS Marketplace (parallel)

### Manual (works without Azure DevOps)

1. Actions → **Trim IDE extension** → `publish=false` → download artifact `trim-ide-vsix`.  
2. https://marketplace.visualstudio.com/manage/publishers/usetrim → **Visual Studio Code** → upload `.vsix`.

### CI (optional)

Secret `VSCE_PAT` = Azure DevOps PAT with Marketplace **Acquire** + **Manage**. Soft-skips when unset (e.g. regions without Azure free/pay-as-you-go).

---

## Local dry-run

```bash
cd extensions/trim-ide
npm ci
npm test
npm run package
# Bootstrap only:
# npx ovsx create-namespace usetrim -p "$OVSX_PAT"
# npx ovsx publish trim-ide-*.vsix -p "$OVSX_PAT"
```

---

## After first public listings

1. Marketplace: `usetrim.trim-ide`  
2. Open VSX: `usetrim.trim-ide`  
3. Docs already point at IDE install; update URLs if store slugs differ.  
4. Prefer real IDE screenshots over placeholder banners when available (`media/screenshot-*.png`).
5. **README images (monorepo):** keep absolute `raw.githubusercontent.com/.../extensions/trim-ide/media/...` URLs (or package with `--baseImagesUrl` pointing at that folder). Plain relative `media/*.png` is rewritten by vsce to the **repo root**, which 404s on Marketplace/Open VSX Details pages.

## Privacy / security links shipped in the VSIX

- [PRIVACY.md](./PRIVACY.md) → https://use-trim.com/privacy  
- [SECURITY.md](./SECURITY.md)  
- [SUPPORT.md](./SUPPORT.md)
