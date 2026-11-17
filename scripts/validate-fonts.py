#!/usr/bin/env python3
"""Cross-check generated fonts with fontTools (development/CI only)."""

from __future__ import annotations

import argparse
import json
import pathlib
import sys

from fontTools.ttLib import TTFont

REQUIRED = {
    "head",
    "hhea",
    "maxp",
    "hmtx",
    "cmap",
    "loca",
    "glyf",
    "name",
    "OS/2",
    "post",
}


def validate(path: pathlib.Path) -> dict:
    font = TTFont(path, lazy=False)
    missing = sorted(REQUIRED - set(font.keys()))
    if missing:
        raise SystemExit(f"{path}: missing tables: {', '.join(missing)}")
    glyphs = font.getGlyphOrder()
    for name in glyphs:
        font["glyf"][name]
    cmap = font.getBestCmap() or {}
    family = font["name"].getDebugName(1) or ""
    style = font["name"].getDebugName(2) or ""
    return {
        "path": str(path),
        "family": family,
        "style": style,
        "glyphs": len(glyphs),
        "cmap": len(cmap),
        "unitsPerEm": font["head"].unitsPerEm,
        "tables": sorted(font.keys()),
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("fonts", nargs="+", type=pathlib.Path)
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()
    reports = []
    for path in args.fonts:
        try:
            reports.append(validate(path))
        except SystemExit as e:
            print(e, file=sys.stderr)
            return 1
        except Exception as e:  # noqa: BLE001 - report the concrete font
            print(f"{path}: {type(e).__name__}: {e}", file=sys.stderr)
            return 1
    if args.json:
        print(json.dumps(reports, ensure_ascii=False, indent=2))
    else:
        for r in reports:
            print(
                f"{r['path']}: {r['family']} {r['style']}, "
                f"{r['glyphs']} glyphs, {r['cmap']} cmap entries, {r['unitsPerEm']} UPM"
            )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
