"""Emit black/white Trim mark masters from the locked transparent blue mark.

Keeps brand/trim-mark-1080.png and brand/trim-mark-transparent-1080.png untouched.
Adds dark (white on #000) + light (black on #FFF) theme pair, matching wordmarks.
Hard-thresholds alpha so soft 3D glow does not leave thin ghost strokes.
"""
from __future__ import annotations

from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
BRAND = ROOT / "brand"
SIZE = 1080
SRC = BRAND / "trim-mark-transparent-1080.png"
DARK_BG = (0, 0, 0, 255)  # #000000
LIGHT_BG = (255, 255, 255, 255)  # #FFFFFF
# Drop soft-glow fringe; keep solid glyph only.
ALPHA_CUT = 140


def mono_glyph(src: Image.Image, *, rgb: tuple[int, int, int]) -> Image.Image:
    arr = np.asarray(src.convert("RGBA"), dtype=np.uint8)
    alpha = arr[:, :, 3].astype(np.float32)
    # Hard mask: opaque core only (no soft rim / highlight ghosts).
    hard = np.where(alpha >= ALPHA_CUT, 255, 0).astype(np.uint8)
    out = np.zeros_like(arr)
    out[:, :, 0] = rgb[0]
    out[:, :, 1] = rgb[1]
    out[:, :, 2] = rgb[2]
    out[:, :, 3] = hard
    return Image.fromarray(out, "RGBA")


def on_bg(glyph: Image.Image, bg: tuple[int, int, int, int]) -> Image.Image:
    canvas = Image.new("RGBA", (SIZE, SIZE), bg)
    if glyph.size != (SIZE, SIZE):
        g = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
        x = (SIZE - glyph.width) // 2
        y = (SIZE - glyph.height) // 2
        g.paste(glyph, (x, y), glyph)
        glyph = g
    canvas.paste(glyph, (0, 0), glyph)
    return canvas


def main() -> None:
    if not SRC.exists():
        raise SystemExit(f"missing locked mark: {SRC}")
    src = Image.open(SRC)
    white = mono_glyph(src, rgb=(255, 255, 255))
    black = mono_glyph(src, rgb=(0, 0, 0))

    on_bg(white, DARK_BG).save(BRAND / "trim-mark-dark-1080.png", "PNG", optimize=True)
    on_bg(black, LIGHT_BG).save(BRAND / "trim-mark-light-1080.png", "PNG", optimize=True)
    white.save(BRAND / "trim-mark-white-transparent-1080.png", "PNG", optimize=True)
    black.save(BRAND / "trim-mark-black-transparent-1080.png", "PNG", optimize=True)

    print("kept trim-mark-1080.png")
    print("kept trim-mark-transparent-1080.png")
    print("wrote trim-mark-dark-1080.png  (white on #000000)")
    print("wrote trim-mark-light-1080.png (black on #FFFFFF)")
    print("wrote trim-mark-white-transparent-1080.png")
    print("wrote trim-mark-black-transparent-1080.png")


if __name__ == "__main__":
    main()
