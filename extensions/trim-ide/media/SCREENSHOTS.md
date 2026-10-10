# Trim IDE - real Marketplace screenshots

Brand banners (`banner-*.png`) stay generated. **Screenshots** should be **live IDE captures** of Trim IDE working end-to-end (not AI mockups).

Marketplace has no separate “screenshot slots”: images live in `README.md` + Getting Started walkthrough media. Target size for our assets: **1280×800** PNG.

## Shot list (required)

| Output file | What to capture (live) | Why |
| --- | --- | --- |
| `screenshot-setup.png` | VS Code / Cursor **Settings** filtered to `trim` - show `Trim: Api Url` = `https://api.use-trim.com`, Auto Start visible | Walkthrough + README “Setup” |
| `screenshot-status.png` | **Output** panel → channel **Trim** after successful chrome sync / Always-on (proxy ready / no fail-closed errors). Optional: status bar Trim item if visible | Walkthrough “working” state |

Optional later (not wired yet): Command Palette with **Trim:** commands; local proxy dashboard `http://127.0.0.1:8888/dashboard`.

## Capture checklist (best practice)

1. Install **Trim IDE** (latest) and `trim` on PATH.
2. Configure for real use (no secrets on screen):
   - `trim.apiUrl` = `https://api.use-trim.com`
   - API key set via **Trim: Set API Key** (never show the key in the shot)
   - Hardware ID registered as agent **IDE**
3. Windows display scaling **100%**; editor zoom default (`window.zoomLevel`: `0` or `1`).
4. Prefer **Dark+** / default dark theme; hide unrelated chat panes if they clutter.
5. Crop tightly to the feature (settings list or Output → Trim). Avoid OS taskbar if easy.
6. No emails, tokens, customer names, or private repo paths in the frame.

## How to capture (Windows)

1. Open the UI for the shot.
2. `Win+Shift+S` → rectangle select → save PNG.
3. Drop files here (exact names):

```text
extensions/trim-ide/media/captures/setup.png
extensions/trim-ide/media/captures/status.png
```

4. Process + overwrite Marketplace assets:

```powershell
cd C:\Users\berek\Downloads\C-Projects\addismender-project\trim
python scripts/_process_ext_screenshots.py
```

5. Review `media/screenshot-setup.png` and `media/screenshot-status.png`.
6. Bump extension patch version, package, republish Marketplace + Open VSX.

## Regenerating brand banners only

```powershell
python scripts/_gen_ext_marketplace_assets.py
```

Does **not** overwrite real screenshots unless you pass `--placeholders`.
