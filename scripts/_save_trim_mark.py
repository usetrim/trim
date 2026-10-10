"""Save NEW inspired mark (materials from inspiration; new silhouette)."""
from __future__ import annotations

from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
BRAND = ROOT / "brand"
SIZE = 1080
BG = (35, 39, 42, 255)
SRC = Path(
    r"C:\Users\berek\.cursor\projects\c-Users-berek-Downloads-C-Projects-addismender-project-trim\assets\trim-mark-inspired-new.jpg"
)


def cut_blue(im: Image.Image) -> Image.Image:
    arr = np.asarray(im.convert("RGBA"), dtype=np.float32)
    r, g, b = arr[:, :, 0], arr[:, :, 1], arr[:, :, 2]
    blue_bias = b - np.maximum(r, g)
    chroma = np.maximum.reduce([np.abs(r - g), np.abs(g - b), np.abs(b - r)])
    keep = (blue_bias > 14) & (b > 48) & (chroma > 16)
    soft = np.clip((blue_bias - 5.0) / 26.0, 0.0, 1.0) * np.clip((chroma - 6.0) / 26.0, 0.0, 1.0)
    alpha = np.where(keep, np.maximum(soft, 0.9), soft * 0.25)
    alpha = np.where(alpha < 0.1, 0.0, alpha)
    out = arr.copy()
    out[:, :, 3] = (np.clip(alpha, 0.0, 1.0) * 255.0).astype(np.uint8)
    return Image.fromarray(out.astype(np.uint8), "RGBA")


def fit(im: Image.Image, *, bg: tuple[int, int, int, int] | None, fill: float) -> Image.Image:
    arr = np.asarray(im)
    ys, xs = np.where(arr[:, :, 3] > 12)
    if len(xs) == 0:
        return Image.new("RGBA", (SIZE, SIZE), bg or (0, 0, 0, 0))
    crop = im.crop((int(xs.min()), int(ys.min()), int(xs.max()) + 1, int(ys.max()) + 1))
    target = int(SIZE * fill)
    w, h = crop.size
    scale = target / max(w, h)
    nw, nh = max(1, int(w * scale)), max(1, int(h * scale))
    crop = crop.resize((nw, nh), Image.Resampling.LANCZOS)
    canvas = Image.new("RGBA", (SIZE, SIZE), bg if bg is not None else (0, 0, 0, 0))
    canvas.paste(crop, ((SIZE - nw) // 2, (SIZE - nh) // 2), crop)
    return canvas


def main() -> None:
    if not SRC.exists():
        raise SystemExit(f"missing source: {SRC}")
    BRAND.mkdir(parents=True, exist_ok=True)
    glyph = cut_blue(Image.open(SRC))
    fit(glyph, bg=BG, fill=0.58).save(BRAND / "trim-mark-1080.png", "PNG", optimize=True)
    fit(glyph, bg=None, fill=0.58).save(BRAND / "trim-mark-transparent-1080.png", "PNG", optimize=True)
    print(f"source={SRC.name}")
    print("wrote trim-mark-1080.png")
    print("wrote trim-mark-transparent-1080.png")


if __name__ == "__main__":
    main()
