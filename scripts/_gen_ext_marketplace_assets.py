"""Generate Trim IDE Marketplace banner assets from brand masters.

By default only banners are written so live IDE screenshots are preserved.
Pass --placeholders to also regenerate placeholder screenshot-*.png panels.

Real screenshots: see extensions/trim-ide/media/SCREENSHOTS.md and
scripts/_process_ext_screenshots.py
"""
from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
BRAND = ROOT / "brand"
MEDIA = ROOT / "extensions" / "trim-ide" / "media"


def load_rgba(path: Path) -> Image.Image:
    return Image.open(path).convert("RGBA")


def crop_alpha(im: Image.Image, pad_ratio: float = 0.0) -> Image.Image:
    arr = np.asarray(im)
    alpha = arr[:, :, 3]
    ys, xs = np.where(alpha > 8)
    if len(xs) == 0:
        return im
    x0, x1 = int(xs.min()), int(xs.max()) + 1
    y0, y1 = int(ys.min()), int(ys.max()) + 1
    cropped = im.crop((x0, y0, x1, y1))
    pad = max(2, int(max(cropped.size) * pad_ratio)) if pad_ratio else 0
    if pad == 0:
        return cropped
    canvas = Image.new("RGBA", (cropped.width + pad * 2, cropped.height + pad * 2), (0, 0, 0, 0))
    canvas.paste(cropped, (pad, pad), cropped)
    return canvas


def fit_square(glyph: Image.Image, size: int, inset: float = 0.1) -> Image.Image:
    cropped = crop_alpha(glyph, 0.0)
    max_inner = max(1, int(round(size * (1.0 - 2.0 * inset)))
    )
    scale = max_inner / max(1, max(cropped.width, cropped.height))
    tw = max(1, int(round(cropped.width * scale)))
    th = max(1, int(round(cropped.height * scale)))
    resized = cropped.resize((tw, th), Image.Resampling.LANCZOS)
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    canvas.paste(resized, ((size - tw) // 2, (size - th) // 2), resized)
    return canvas


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--placeholders",
        action="store_true",
        help="Also regenerate placeholder screenshot-*.png (overwrites live captures)",
    )
    args = parser.parse_args()

    black = load_rgba(BRAND / "trim-mark-black-transparent-1080.png")
    white = load_rgba(BRAND / "trim-mark-white-transparent-1080.png")
    MEDIA.mkdir(parents=True, exist_ok=True)

    for name, bg, glyph in (
        ("banner-dark.png", (15, 15, 18, 255), white),
        ("banner-light.png", (248, 248, 250, 255), black),
    ):
        banner = Image.new("RGBA", (1280, 640), bg)
        mark = fit_square(glyph, 280, inset=0.08)
        banner.paste(mark, ((1280 - 280) // 2, (640 - 280) // 2 - 40), mark)
        path = MEDIA / name
        banner.convert("RGB").save(path, format="PNG", optimize=True)
        print(f"wrote {path}")

    if args.placeholders:
        for name, bg, glyph in (
            ("screenshot-setup.png", (248, 248, 250, 255), black),
            ("screenshot-status.png", (15, 15, 18, 255), white),
        ):
            shot = Image.new("RGBA", (1280, 800), bg)
            mark = fit_square(glyph, 160, inset=0.1)
            shot.paste(mark, (80, 80), mark)
            panel = Image.new(
                "RGBA",
                (1120, 480),
                (255, 255, 255, 18) if bg[0] < 128 else (0, 0, 0, 12),
            )
            shot.paste(panel, (80, 280), panel)
            path = MEDIA / name
            shot.convert("RGB").save(path, format="PNG", optimize=True)
            print(f"wrote {path} (placeholder)")
    else:
        print("skipped screenshot-*.png (use --placeholders or scripts/_process_ext_screenshots.py)")

    print("done")


if __name__ == "__main__":
    main()
