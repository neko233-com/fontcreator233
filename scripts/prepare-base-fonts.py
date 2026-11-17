#!/usr/bin/env python3
"""Prepare the embedded OFL base fonts used by fontcreator233.

Sources (Google Fonts, main branch, retrieved 2026-11-17):
  SC: https://github.com/google/fonts/raw/main/ofl/notosanssc/NotoSansSC%5Bwght%5D.ttf
  KR: https://github.com/google/fonts/raw/main/ofl/notosanskr/NotoSansKR%5Bwght%5D.ttf
  OFL: https://github.com/google/fonts/raw/main/ofl/notosanssc/OFL.txt

Noto Sans SC/KR are SIL OFL 1.1 fonts. Their variable fonts default to
wght=100 (Thin), and x/image/font/sfnt does not apply gvar deltas. This
script therefore:

  1. downloads the variable fonts (or accepts local --sc-source/--kr-source),
  2. instantiates the wght axis at 400 (Regular),
  3. subsets SC to its full cmap and KR to Hangul/Jamo/compat/fullwidth,
  4. strips hinting and layout tables (only outlines/cmap/metrics are used),
  5. renames all branding to fontcreator233 Base SC/KR (the OFL reserves
     the name "Source"), keeps copyright + OFL text + attribution,
  6. sets deterministic head timestamps and post format 3.0,
  7. writes assets/base/ and prints SHA-256 values.

Runtime fontcreator233 never runs this script and never downloads anything.

Usage:
  python scripts/prepare-base-fonts.py --sc-source NotoSansSC.ttf --kr-source NotoSansKR.ttf
  python scripts/prepare-base-fonts.py                # downloads sources
"""

from __future__ import annotations

import argparse
import hashlib
import pathlib
import sys
import urllib.request
from datetime import datetime, timezone

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

SOURCES = {
    "sc": "https://github.com/google/fonts/raw/main/ofl/notosanssc/NotoSansSC%5Bwght%5D.ttf",
    "kr": "https://github.com/google/fonts/raw/main/ofl/notosanskr/NotoSansKR%5Bwght%5D.ttf",
    "ofl": "https://github.com/google/fonts/raw/main/ofl/notosanssc/OFL.txt",
}

SOURCE_SHA256 = {
    "sc": "a3041811a78c361b1de50f953c805e0244951c21c5bd412f7232ef0d899af0da",
    "kr": "194018e6b2b293a7964f037b25c0249ce1418bc9ab3c971060a03aa57861e252",
}

MAC_EPOCH = datetime(1904, 1, 1, tzinfo=timezone.utc)
FIXED_DATE = datetime(2026, 1, 1, tzinfo=timezone.utc)
FIXED_MAC_SECONDS = int((FIXED_DATE - MAC_EPOCH).total_seconds())

OFL_TEXT = "This Font Software is licensed under the SIL Open Font License, Version 1.1. This license is available with a FAQ at: https://scripts.sil.org/OFL"
OFL_URL = "https://scripts.sil.org/OFL"
COPYRIGHT = (
    "Copyright 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name "
    "'Source'. Derived from Noto Sans SC/KR for fontcreator233 and renamed per "
    "the SIL Open Font License 1.1."
)

KR_UNICODES = (
    list(range(0x1100, 0x1200))      # Hangul Jamo
    + list(range(0x3000, 0x3040))    # CJK punctuation
    + list(range(0x3130, 0x3190))    # Hangul compatibility Jamo
    + list(range(0xA960, 0xA980))    # Hangul Jamo Extended-A
    + list(range(0xAC00, 0xD7A4))    # Hangul Syllables
    + list(range(0xD7B0, 0xD800))    # Hangul Jamo Extended-B
    + list(range(0xFFA0, 0xFFDD))    # Halfwidth Hangul
)

REPLACED_NAME_IDS = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 16, 17}


def sha256(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def download(url: str, dest: pathlib.Path) -> pathlib.Path:
    print(f"downloading {url}")
    dest.parent.mkdir(parents=True, exist_ok=True)
    with urllib.request.urlopen(url, timeout=120) as r, dest.open("wb") as f:
        while True:
            chunk = r.read(1 << 20)
            if not chunk:
                break
            f.write(chunk)
    return dest


def set_name(font: TTFont, nid: int, value: str) -> None:
    name = font["name"]
    name.names = [n for n in name.names if n.nameID != nid]
    name.setName(value, nid, 3, 1, 0x409)
    name.setName(value, nid, 1, 0, 0)


def rename(font: TTFont, family: str, subfamily: str, ps_name: str) -> None:
    name = font["name"]
    name.names = [n for n in name.names if n.nameID not in REPLACED_NAME_IDS]
    set_name(font, 0, COPYRIGHT)
    set_name(font, 1, family)
    set_name(font, 2, subfamily)
    set_name(font, 3, f"{family} {subfamily} 1.000;2026")
    set_name(font, 4, f"{family} {subfamily}")
    set_name(font, 5, "Version 1.000")
    set_name(font, 6, ps_name)
    set_name(font, 8, "fontcreator233")
    set_name(font, 9, "fontcreator233")
    set_name(font, 10, f"Static base font derived from Noto for {family}; OFL-1.1.")
    set_name(font, 11, "https://github.com/neko233-com/fontcreator233")
    set_name(font, 12, "https://github.com/neko233-com/fontcreator233")
    set_name(font, 13, OFL_TEXT)
    set_name(font, 14, OFL_URL)
    set_name(font, 16, family)
    set_name(font, 17, subfamily)


def instantiate(source: pathlib.Path) -> TTFont:
    font = TTFont(source)
    if "fvar" in font:
        font = instantiateVariableFont(
            font, {"wght": 400}, updateFontNames=True, inplace=False
        )
    return font


def subset_font(font: TTFont, unicodes: list[int]) -> None:
    opts = subset.Options()
    opts.hinting = False
    opts.layout_features = []
    opts.name_IDs = ["*"]
    opts.name_legacy = True
    opts.name_languages = ["*"]
    opts.notdef_outline = True
    opts.recalc_bounds = True
    opts.recalc_timestamp = False
    opts.drop_tables += [
        "BASE",
        "GDEF",
        "GPOS",
        "GSUB",
        "STAT",
        "vhea",
        "vmtx",
        "gasp",
        "prep",
        "DSIG",
    ]
    opts.unicodes = sorted(set(unicodes))
    sub = subset.Subsetter(options=opts)
    sub.populate(unicodes=opts.unicodes)
    sub.subset(font)


def finalize(font: TTFont) -> None:
    font["head"].created = FIXED_MAC_SECONDS
    font["head"].modified = FIXED_MAC_SECONDS
    font["OS/2"].usWeightClass = 400
    font["OS/2"].usWidthClass = 5
    font["OS/2"].fsSelection = (font["OS/2"].fsSelection & ~0x21) | 0x40
    font["head"].macStyle = 0
    if "post" in font:
        font["post"].formatType = 3.0
        font["post"].italicAngle = 0.0


def build(
    source: pathlib.Path,
    out_dir: pathlib.Path,
    filename: str,
    unicodes: list[int] | None,
    family: str,
    subfamily: str,
    ps_name: str,
) -> pathlib.Path:
    font = instantiate(source)
    if unicodes is None:
        subset_font(font, list(font.getBestCmap().keys()))
    else:
        subset_font(font, unicodes)
    rename(font, family, subfamily, ps_name)
    finalize(font)
    out_dir.mkdir(parents=True, exist_ok=True)
    out = out_dir / filename
    font.save(out)
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--sc-source", type=pathlib.Path)
    ap.add_argument("--kr-source", type=pathlib.Path)
    ap.add_argument("--ofl-source", type=pathlib.Path)
    ap.add_argument("--out-dir", type=pathlib.Path, default=pathlib.Path("assets/base"))
    ap.add_argument("--force", action="store_true", help="overwrite existing outputs")
    ap.add_argument(
        "--skip-sha-check",
        action="store_true",
        help="do not fail when downloaded source hashes differ",
    )
    args = ap.parse_args()

    out_dir: pathlib.Path = args.out_dir
    if not args.force and (out_dir / "Fontcreator233BaseSC-Regular.ttf").exists():
        print(f"{out_dir} already contains base fonts; pass --force to regenerate")
        return 2

    tmp = out_dir.parent / ".base-font-sources"
    sc_src = args.sc_source or download(SOURCES["sc"], tmp / "NotoSansSC.ttf")
    kr_src = args.kr_source or download(SOURCES["kr"], tmp / "NotoSansKR.ttf")
    ofl_path = args.ofl_source or download(SOURCES["ofl"], tmp / "OFL.txt")

    for label, path in (("sc", sc_src), ("kr", kr_src)):
        got = sha256(path)
        want = SOURCE_SHA256[label]
        print(f"{label} source sha256 {got}")
        if got != want and not args.skip_sha_check:
            print(
                f"source {label} hash changed (want {want}); "
                "use --skip-sha-check if this is an intentional upstream update",
                file=sys.stderr,
            )
            return 1

    sc_out = build(
        sc_src,
        out_dir,
        "Fontcreator233BaseSC-Regular.ttf",
        None,
        "fontcreator233 Base SC",
        "Regular",
        "Fontcreator233BaseSC-Regular",
    )
    kr_out = build(
        kr_src,
        out_dir,
        "Fontcreator233BaseKR-Regular.ttf",
        KR_UNICODES,
        "fontcreator233 Base KR",
        "Regular",
        "Fontcreator233BaseKR-Regular",
    )
    ofl_out = out_dir / "OFL.txt"
    ofl_out.write_bytes(ofl_path.read_bytes())

    for p in (sc_out, kr_out, ofl_out):
        print(f"{p}: {p.stat().st_size} bytes sha256={sha256(p)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
