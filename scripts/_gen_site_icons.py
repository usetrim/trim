"""
Generate page logos + site icons from brand mark masters.

Masters in brand/ are NEVER modified.

  brand/trim-mark-black-transparent-1080.png → black glyph
  brand/trim-mark-white-transparent-1080.png → white glyph

Output (web + admin public):

  public/brand/trim-mark-black.png  → black mark (light UI)
  public/brand/trim-mark-white.png  → white mark (dark UI)

Favicons / PWA icons: same mark on transparent square (except apple-touch,
which stays opaque light tile for iOS).
"""
from __future__ import annotations

import io
import struct
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
BRAND = ROOT / "brand"
OUT_ROOTS = [
    ROOT / "apps" / "web" / "public",
    ROOT / "apps" / "admin" / "public",
]
# VS Code / Cursor extension marketplace + theme-aware media (keep in sync with web/admin).
EXT_MEDIA = ROOT / "extensions" / "trim-ide" / "media"
EXT_ICON_SIZE = 128
ICON_SIZES = [16, 32, 48, 180, 192, 512]
# Page mark export height (width follows glyph bbox).
MARK_HEIGHT = 256
OLD_PAGE_LOGOS = (
    "trim-wordmark-black.png",
    "trim-wordmark-white.png",
)


def load_glyph(path: Path) -> Image.Image:
    """Load locked transparent master; keep alpha, force pure black or white RGB."""
    im = Image.open(path).convert("RGBA")
    arr = np.asarray(im, dtype=np.uint8)
    alpha = arr[:, :, 3]
    # Infer intended ink from non-transparent luminance.
    lum = (
        0.299 * arr[:, :, 0].astype(np.float32)
        + 0.587 * arr[:, :, 1].astype(np.float32)
        + 0.114 * arr[:, :, 2].astype(np.float32)
    )
    mask = alpha > 8
    if not np.any(mask):
        raise SystemExit(f"empty glyph: {path}")
    mean_lum = float(lum[mask].mean())
    ink = 255 if mean_lum >= 128 else 0
    out = np.zeros_like(arr)
    out[:, :, 0:3] = ink
    out[:, :, 3] = alpha
    return Image.fromarray(out, "RGBA")


def crop_to_alpha(im: Image.Image, pad_ratio: float = 0.08) -> Image.Image:
    alpha = np.asarray(im.split()[-1])
    ys, xs = np.where(alpha > 8)
    if len(xs) == 0:
        return im
    x0, x1 = int(xs.min()), int(xs.max()) + 1
    y0, y1 = int(ys.min()), int(ys.max()) + 1
    cropped = im.crop((x0, y0, x1, y1))
    pad = max(2, int(max(cropped.size) * pad_ratio))
    canvas = Image.new("RGBA", (cropped.width + pad * 2, cropped.height + pad * 2), (0, 0, 0, 0))
    canvas.paste(cropped, (pad, pad), cropped)
    return canvas


def fit_square_transparent(
    glyph: Image.Image,
    size: int,
    *,
    edge_inset: float = 0.04,
) -> Image.Image:
    """Mark on transparent square, filled nearly to edges for tab readability."""
    cropped = crop_to_alpha(glyph, pad_ratio=0.0)
    max_inner = max(1, int(round(size * (1.0 - 2.0 * edge_inset))))
    if size <= 32:
        max_inner = max(1, size - (2 if size >= 32 else 1))
    scale = max_inner / max(1, max(cropped.width, cropped.height))
    target_w = max(1, int(round(cropped.width * scale)))
    target_h = max(1, int(round(cropped.height * scale)))
    if target_w > size or target_h > size:
        scale = min(size / max(1, cropped.width), size / max(1, cropped.height))
        target_w = max(1, int(round(cropped.width * scale)))
        target_h = max(1, int(round(cropped.height * scale)))
    resized = cropped.resize((target_w, target_h), Image.Resampling.LANCZOS)
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    canvas.paste(resized, ((size - target_w) // 2, (size - target_h) // 2), resized)
    return canvas


def opaque_light_tile(black_glyph: Image.Image, size: int) -> Image.Image:
    """Apple-touch: black mark on opaque white (iOS rejects transparent tiles)."""
    sq = fit_square_transparent(black_glyph, size, edge_inset=0.12)
    bg = Image.new("RGB", (size, size), (255, 255, 255))
    bg.paste(sq, mask=sq.split()[-1])
    return bg


def save_page_mark(glyph: Image.Image, path: Path, height: int = MARK_HEIGHT) -> tuple[int, int]:
    cropped = crop_to_alpha(glyph, pad_ratio=0.06)
    w = max(1, int(round(cropped.width * (height / cropped.height))))
    out = cropped.resize((w, height), Image.Resampling.LANCZOS)
    path.parent.mkdir(parents=True, exist_ok=True)
    out.save(path, format="PNG", optimize=True)
    return out.size


def save_ico(path: Path, images: dict[int, Image.Image]) -> None:
    ordered = [16, 32, 48]
    pngs: list[tuple[int, bytes]] = []
    for s in ordered:
        buf = io.BytesIO()
        images[s].convert("RGBA").save(buf, format="PNG", optimize=True)
        pngs.append((s, buf.getvalue()))
    count = len(pngs)
    header = struct.pack("<HHH", 0, 1, count)
    offset = 6 + 16 * count
    entries = bytearray()
    blob = bytearray()
    for s, png in pngs:
        w = 0 if s >= 256 else s
        h = 0 if s >= 256 else s
        entries += struct.pack("<BBBBHHII", w, h, 0, 0, 1, 32, len(png), offset)
        offset += len(png)
        blob += png
    path.write_bytes(header + entries + blob)


def remove_old_page_logos(root: Path) -> None:
    for name in OLD_PAGE_LOGOS:
        for folder in (root / "brand", root / "icons"):
            path = folder / name
            if path.exists():
                path.unlink()
                print(f"removed {path}")


def main() -> None:
    black_src = BRAND / "trim-mark-black-transparent-1080.png"
    white_src = BRAND / "trim-mark-white-transparent-1080.png"
    if not black_src.exists() or not white_src.exists():
        raise SystemExit(
            "missing mark masters: brand/trim-mark-{black,white}-transparent-1080.png"
        )

    black_glyph = load_glyph(black_src)
    white_glyph = load_glyph(white_src)

    dims: dict[str, tuple[int, int]] = {}
    for root in OUT_ROOTS:
        brand_dir = root / "brand"
        icons = root / "icons"
        icons.mkdir(parents=True, exist_ok=True)
        brand_dir.mkdir(parents=True, exist_ok=True)

        remove_old_page_logos(root)

        bw = save_page_mark(black_glyph, brand_dir / "trim-mark-black.png")
        ww = save_page_mark(white_glyph, brand_dir / "trim-mark-white.png")
        dims["black"] = bw
        dims["white"] = ww
        print(f"wrote {brand_dir / 'trim-mark-black.png'} {bw}")
        print(f"wrote {brand_dir / 'trim-mark-white.png'} {ww}")

        # Also expose under /icons for consistency with older paths.
        save_page_mark(black_glyph, icons / "trim-mark-black.png")
        save_page_mark(white_glyph, icons / "trim-mark-white.png")

        for theme, glyph in (("dark", white_glyph), ("light", black_glyph)):
            theme_pngs: dict[int, Image.Image] = {}
            for s in ICON_SIZES:
                inset = 0.04 if s <= 48 else 0.08
                img = fit_square_transparent(glyph, s, edge_inset=inset)
                path = icons / f"icon-{s}-{theme}.png"
                img.save(path, format="PNG", optimize=True)
                theme_pngs[s] = img
                print(f"wrote {path} ({path.stat().st_size} B)")
            ico_path = icons / f"favicon-{theme}.ico"
            save_ico(ico_path, theme_pngs)
            print(f"wrote {ico_path} ({ico_path.stat().st_size} B)")

        apple = opaque_light_tile(black_glyph, 180)
        apple.save(icons / "apple-touch-icon.png", format="PNG", optimize=True)
        apple.save(root / "apple-touch-icon.png", format="PNG", optimize=True)

        # Default favicon = light-mode (black mark, transparent)
        save_ico(
            root / "favicon.ico",
            {
                16: fit_square_transparent(black_glyph, 16, edge_inset=0.0),
                32: fit_square_transparent(black_glyph, 32, edge_inset=0.0),
                48: fit_square_transparent(black_glyph, 48, edge_inset=0.04),
            },
        )
        print(f"wrote {root / 'favicon.ico'}")

    # --- Trim IDE extension (marketplace icon + light/dark marks) ---
    EXT_MEDIA.mkdir(parents=True, exist_ok=True)
    # Marketplace allows one icon (128×128). Use black mark on opaque light tile so it
    # stays readable on both light and dark Extensions sidebars / Marketplace cards.
    market = opaque_light_tile(black_glyph, EXT_ICON_SIZE)
    market_path = EXT_MEDIA / "icon.png"
    market.save(market_path, format="PNG", optimize=True)
    print(f"wrote {market_path}")

    # Theme-aware pair (same assets as web/admin brand marks, square 128 for IDE chrome).
    light_icon = fit_square_transparent(black_glyph, EXT_ICON_SIZE, edge_inset=0.08)
    dark_icon = fit_square_transparent(white_glyph, EXT_ICON_SIZE, edge_inset=0.08)
    light_path = EXT_MEDIA / "trim-icon-light.png"
    dark_path = EXT_MEDIA / "trim-icon-dark.png"
    light_icon.save(light_path, format="PNG", optimize=True)
    dark_icon.save(dark_path, format="PNG", optimize=True)
    print(f"wrote {light_path}")
    print(f"wrote {dark_path}")

    # Also keep page-mark crops for README / docs screenshots.
    save_page_mark(black_glyph, EXT_MEDIA / "trim-mark-black.png", height=128)
    save_page_mark(white_glyph, EXT_MEDIA / "trim-mark-white.png", height=128)
    print(f"wrote {EXT_MEDIA / 'trim-mark-black.png'}")
    print(f"wrote {EXT_MEDIA / 'trim-mark-white.png'}")

    readme = EXT_MEDIA / "README.md"
    readme.write_text(
        "# Trim IDE media\n\n"
        "Generated by `scripts/_gen_site_icons.py` from `brand/trim-mark-*-transparent-1080.png` "
        "(same masters as web/admin).\n\n"
        "- `icon.png` - VS Code Marketplace / Extensions list (`package.json` `icon`, 128×128, "
        "black mark on light tile)\n"
        "- `trim-icon-light.png` / `trim-icon-dark.png` - transparent square marks for theme-aware "
        "UI (activity bar / views when contributed)\n"
        "- `trim-mark-black.png` / `trim-mark-white.png` - page-mark crops (light / dark UI)\n\n"
        "Do not hand-edit; re-run the generator after brand master updates.\n",
        encoding="utf-8",
    )
    print(f"wrote {readme}")

    if dims.get("black"):
        w, h = dims["black"]
        aspect = w / h
        hint = ROOT / "scripts" / "_mark_aspect.txt"
        hint.write_text(f"{aspect:.6f}\n{w}x{h}\n", encoding="utf-8")
        # Keep legacy filename in sync for any old readers.
        (ROOT / "scripts" / "_wordmark_aspect.txt").write_text(
            f"{aspect:.6f}\n{w}x{h}\n", encoding="utf-8"
        )
        print(f"aspect {aspect:.6f} ({w}x{h})")

    print("done")


if __name__ == "__main__":
    main()
