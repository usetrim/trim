# Trim brand assets

## Mark (icon + page logo master)

Canonical mark only (**PNG**, 1080×1080). Same silhouette; color variants.

Pure black/white for theme pair (not zinc grays).

| File | Use |
| --- | --- |
| [`trim-mark-dark-1080.png`](./trim-mark-dark-1080.png) | Dark - **white** `#FFFFFF` on **black** `#000000` |
| [`trim-mark-light-1080.png`](./trim-mark-light-1080.png) | Light - **black** `#000000` on **white** `#FFFFFF` |
| [`trim-mark-white-transparent-1080.png`](./trim-mark-white-transparent-1080.png) | White cutout (transparent) |
| [`trim-mark-black-transparent-1080.png`](./trim-mark-black-transparent-1080.png) | Black cutout (transparent) |

**Do not edit these masters for app delivery.** Page logos and favicons are
generated into `apps/*/public/brand` and `apps/*/public/icons` via
`python scripts/_gen_site_icons.py` (mark only, transparent background).

Rebuild B/W masters from a color transparent source (if present):

```bash
python scripts/_save_trim_mark_bw.py
```
