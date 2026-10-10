# npm distribution (`@usetrim/trim`)

Thin npm package that installs the **official Trim CLI binary** from GitHub Releases
(with SHA-256 verification). This is the industry-standard “binary via npm” path for
Go CLIs used by JS/AI developers (`npm install -g` / `npx`).

## Layout

```text
packaging/npm/@usetrim/trim/
  package.json      # name @usetrim/trim, bin trim
  bin/trim.js       # shim + self-heal
  scripts/install.js
  scripts/platform.js
  README.md
```

## Publish (Trusted Publishing - best practice)

CI: `.github/workflows/publish-npm.yml`

1. Org **`usetrim`** + account **2FA** on https://www.npmjs.com  
2. **First publish only:** package must exist before OIDC can be configured  
   - Either push a `v*` tag with repo secret `NPM_TOKEN` once, or  
   - `cd packaging/npm/@usetrim/trim && npm login && npm publish --access public`  
3. On the package page → **Trusted Publisher** → GitHub Actions:
   - Org/user: `usetrim`
   - Repo: `trim`
   - Workflow file: `publish-npm.yml`
   - Allow **`npm publish`**
4. Later tags use OIDC (`permissions.id-token: write`). Delete `NPM_TOKEN` after that works.
5. Optional harden: package publishing access → disallow classic tokens.

See `packaging/DISTRIBUTION-LAUNCH.md` for the full checklist.

## Local verify (no publish)

```bash
cd packaging/npm/@usetrim/trim
node --test scripts/platform.test.js
npm pack
# In a temp dir:
#   npm install -g ./usetrim-trim-0.1.0.tgz
#   trim version
```

## Why not optionalDependencies platform packages (yet)?

Best-in-class long term is per-OS packages under `optionalDependencies` (Biome / esbuild style).
That requires publishing N platform packages per release from `dist/`.  
This wrapper ships **today** against existing Releases; we can migrate to optional-deps later without changing the `@usetrim/trim` user command.
