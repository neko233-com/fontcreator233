# Latin starter outline packs

These packs make the four professional family templates in `templates/`
usable out of the box:

| Pack | Template | Source | Weight |
|---|---|---|---|
| `hei` | `hei.recipe.yaml` | Noto Sans Regular 2.015 | Regular 400 |
| `song` | `song.recipe.yaml` | Noto Serif Regular 2.015 | Regular 400 |
| `puhui` | `puhuiti.recipe.yaml` | Poppins Regular 4.004 | Regular 400 |
| `mono` | `mono.recipe.yaml` | JetBrains Mono Regular 2.305 | Regular 600-unit mono |

Each directory contains 94 ASCII glyphs (`U+0020`–`U+007E`, space excluded)
exported by:

```bash
fontcreator import font <source.ttf> --out <dir> \
  --extract-glyphs --unicodes 0020-007E
```

## Why static?

The overrides are real outlines with their own advances, not skeleton
centerlines. The recipe's `weight`, `width`, and `variable` settings do not
change them. These templates are deliberately `variable: false`; use the
parametric recipes (`product-ui`, `poster-display`, `editorial-serif`, etc.)
when you need another weight or width, or import the matching source weight
into a separate pack.

## Licensing and naming

All four sources are SIL OFL 1.1. The full text is in `OFL.txt`; attribution is
in `NOTICE.md` at the repository root. Generated fonts must remain OFL-1.1 and
must not use the source family names (`Noto`, `Source`, `Poppins`,
`JetBrains Mono`).

Alibaba PuHuiTi is **not** included. Its license forbids modifying or
redistributing the font files, so `puhuiti` uses Poppins as a geometric,
single-storey-a/g starter instead. If you have a licensed Alibaba PuHuiTi TTF,
import it yourself and point the recipe's `glyphs.overrides` at the exported
glyphs.
