# Agent workflow: from reference image to font

fontcreator233 assumes the agent can see. The CLI is deterministic and offline;
it does not inspect images, call a model, or guess a style.

## 1. Read the reference like a type designer

Write a short style brief before touching the recipe:

- **Skeleton**: geometric, humanist, or serif. Look at bowls, diagonals, and
  whether the lowercase `a` is single- or double-story.
- **Weight**: measure stroke thickness relative to x-height.
- **Contrast**: difference between vertical stems and horizontal hairlines.
- **Width**: condensed, normal, or extended; compare cap height to advance.
- **Terminals**: flat-cut, rounded, angled, or flared.
- **Serifs**: none, slab, bracketed, or hairline.
- **Slant**: 0° upright, 6–12° italic.
- **Texture**: clean, inked, rough, hand-drawn, or poster-printed.
- **Palette**: use it for `meta.accent` and the specimen, not for outline
  geometry.

## 2. Mapping table

| Visual | Recipe |
|---|---|
| Thin geometric | `weight: 250`, `contrast: 0.05`, `terminal: round` |
| Neo-grotesk text | `weight: 450`, `contrast: 0.08`, `width: 100` |
| Heavy poster | `weight: 780`, `contrast: 0.15`, `width: 108` |
| Didone | `weight: 480`, `contrast: 0.9`, `serif: hairline`, `serifSize: 0.5` |
| Slab | `weight: 560`, `contrast: 0.2`, `serif: slab`, `serifSize: 0.7` |
| Humanist book | `skeleton: humanist`, `weight: 420`, `contrast: 0.3` |
| Art Nouveau | `contrast: 0.62`, `terminal: flare`, `roughness: 0.18` |
| Brush / ink | `weight: 620`, `terminal: flare`, `roughness: 0.35`, `seed: 233` |
| Condensed label | `width: 84`, `weight: 620` |
| Extended display | `width: 118`, `weight: 650` |
| Italic poster | `slant: 9`, resolves style to `Italic` |

`seed` controls all deterministic jitter. Keep it fixed while comparing
parameter changes.

## 3. Three worked examples

### Neon poster → Aurora Grotesk

Reference: dark poster, wide geometric caps, uniform tubes, rounded ends.

```yaml
style:
  skeleton: geometric
  weight: 620
  width: 104
  contrast: 0.06
  terminal: round
  serif: none
  seed: 233
```

### Fashion editorial → Noir Didone

Reference: high-contrast serif, narrow caps, sharp hairline serifs, slanted
italic.

```yaml
meta:
  style: Noir
style:
  skeleton: serif
  weight: 500
  width: 94
  slant: 8
  contrast: 0.86
  terminal: angled
  serif: hairline
  serifSize: 0.6
```

### Botanical Art Nouveau → Botanic Display

Reference: organic stems, flared terminals, medium contrast, slight rotation.

```yaml
style:
  skeleton: geometric
  weight: 540
  width: 108
  contrast: 0.42
  terminal: flare
  tracking: 1.5
  roughness: 0.08
  jitterRotation: 0.2
  seed: 233
```

## 4. Learn from a reference font

When the user supplies an existing TTF/OTF instead of only an image:

```bash
fontcreator learn reference.ttf --out work/learned --json
```

Read `font-report.md` for name records, vertical metrics, cmap coverage,
GSUB/GPOS feature tags and lookup types, legacy kern pairs, variable axes, color
tables, and sampled glyph statistics. The generated `recipe.yaml` is a starting
point. Do not redistribute or copy a proprietary font; use the report to
understand proportions and features, then make new outlines.

## 5. Build and review loop

```bash
fontcreator validate work/font.recipe.yaml --json
fontcreator build work/font.recipe.yaml --out dist --target all --json --no-open
fontcreator specimen dist/<font>.ttf --open
fontcreator inspect dist/<font>.ttf --json
```

- Review the specimen at large sizes and in paragraphs.
- Check `coverage` and `coverageReport`; warnings for missing CJK/Hangul/kana
  are real.
- If a single glyph is wrong, export it with `library export`, edit the SVG,
  and set `glyphs.overrides`.
- Keep the previous build; workspace state cleans only files it generated and
  verifies by SHA-256.

## 6. Shipping

- Unity TMP: install the generated `unity/` package, run
  `Tools/Fontcreator233/Create TMP Font Assets`, then use the fallback menu for
  split CJK fonts.
- Unreal: run `unreal/Scripts/fontcreator233_import.py`; configure
  `CompositeFont` ranges from the manifest if automatic creation is unavailable.
- Web: use `web/font.css`; split fonts already carry `unicode-range`.
- System: follow `system/INSTALL.md`.

## 7. Limits to state honestly

The font does not support Arabic, Hebrew, Indic scripts, Thai, vertical layout,
complex shaping, color fonts, or variable axes other than one `wght` axis.
Generated OpenType covers `liga`, `kern`, `case`, `frac`, `tnum`, `ss01`,
`smcp`, `onum`, `zero`, `sups`, `subs`, `ccmp`, simple top-center `mark`
anchors, and `mkmk` stacked marks. With `output.variable: true`, one real `wght`
axis is written via `fvar` + `gvar` for Latin/Greek/Cyrillic/symbol glyphs; CJK,
Hangul, and kana stay static, and no `HVAR`/`MVAR` or additional axes are
generated. Recipe-defined `calt` exact-sequence rules and opt-in `curs`
cursive attachment (`output.cursive: true`) are real GPOS/GSUB lookups; advanced
`cswh` and contextual classes are not generated. CJK/Hangul outlines come from the embedded OFL
bases and receive geometric restyling and synthetic bold, not a full redraw.
`full` coverage means “all glyphs present in the embedded bases”, not every
Unicode block.
