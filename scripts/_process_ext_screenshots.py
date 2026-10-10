"""Resize/crop live IDE captures into Trim IDE Marketplace screenshot slots.

Drop captures at:
  extensions/trim-ide/media/captures/setup.png
  extensions/trim-ide/media/captures/status.png

Writes:
  extensions/trim-ide/media/screenshot-setup.png   (1280x800)
  extensions/trim-ide/media/screenshot-status.png  (1280x800)
"""
from __future__ import annotations

import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
MEDIA = ROOT / "extensions" / "trim-ide" / "media"
CAPTURES = MEDIA / "captures"
OUT_W, OUT_H = 1280, 800

MAP = {
    "setup.png": "screenshot-setup.png",
    "status.png": "screenshot-status.png",
}


def contain_pad(im: Image.Image, tw: int, th: int, fill: tuple[int, int, int] = (15, 15, 18)) -> Image.Image:
    """Scale to fit entirely inside target; pad (letterbox). Keeps UI text readable."""
    src = im.convert("RGB")
    sw, sh = src.size
    if sw < 1 or sh < 1:
        raise ValueError("empty image")
    scale = min(tw / sw, th / sh)
    nw = max(1, int(round(sw * scale)))
    nh = max(1, int(round(sh * scale)))
    resized = src.resize((nw, nh), Image.Resampling.LANCZOS)
    canvas = Image.new("RGB", (tw, th), fill)
    canvas.paste(resized, ((tw - nw) // 2, (th - nh) // 2))
    return canvas


def main() -> int:
    CAPTURES.mkdir(parents=True, exist_ok=True)
    missing = [name for name in MAP if not (CAPTURES / name).is_file()]
    if missing:
        print("Missing capture file(s):", ", ".join(missing), file=sys.stderr)
        print(f"Place PNGs under {CAPTURES}", file=sys.stderr)
        print("See extensions/trim-ide/media/SCREENSHOTS.md", file=sys.stderr)
        return 1

    for src_name, dest_name in MAP.items():
        src = CAPTURES / src_name
        dest = MEDIA / dest_name
        # Dark pad matches galleryBanner / Output theme; avoids center-crop cutting labels.
        out = contain_pad(Image.open(src), OUT_W, OUT_H)
        out.save(dest, format="PNG", optimize=True)
        print(f"wrote {dest} ({OUT_W}x{OUT_H}) from {src.name}")

    print("done - review screenshots, then bump trim-ide version and republish")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
