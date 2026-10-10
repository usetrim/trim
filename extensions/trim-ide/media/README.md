# Trim IDE media

| File | Source |
| --- | --- |
| `icon.png`, `trim-icon-*.png`, `trim-mark-*.png` | `scripts/_gen_site_icons.py` from `brand/trim-mark-*-transparent-1080.png` |
| `banner-*.png` | `scripts/_gen_ext_marketplace_assets.py` (brand banners) |
| `screenshot-*.png` | **Live IDE captures** via `media/SCREENSHOTS.md` + `scripts/_process_ext_screenshots.py` |

`package.json` `icon` → `icon.png` (128×128). Walkthrough / README use banners and screenshots.

Do not hand-edit brand glyphs. Prefer real screenshots over `--placeholders`.
