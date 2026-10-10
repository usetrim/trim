# Site icons (generated)

Trim mark (**transparent** background - no color tile).
Derived from `brand/trim-mark-{black,white}-transparent-1080.png` (masters untouched).

| File pattern | When |
| --- | --- |
| `icon-*-dark.png` | Dark UI - **white** mark, transparent |
| `icon-*-light.png` | Light UI - **black** mark, transparent |
| `apple-touch-icon.png` | iOS home - black mark on **opaque white** (required) |
| `favicon-*.ico` | Legacy tab icon (16/32/48, transparent) |
| `../favicon.ico` | Default (black mark) |
| `trim-mark-*.png` | Same mark assets as `/brand/` |

Page logos live in `/brand/trim-mark-{black,white}.png`.

```bash
python scripts/_gen_site_icons.py
```
