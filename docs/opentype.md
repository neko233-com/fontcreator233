# OpenType features

fontcreator233 writes real OpenType layout tables into every generated TTF/WOFF.
The specimen exposes the features live; fontTools asserts the tables in tests and
HarfBuzz shaping was used for manual verification.

## Generated tables

| Table | Purpose |
|---|---|
| `GSUB` | Ligatures and one-for-one substitutions |
| `GPOS` | Pair kerning, mark-to-base, and mark-to-mark attachment |
| `GDEF` | Glyph classes for marks and alternates |
| `kern` | Legacy pair kerning for older software |
| `fvar` | Opt-in `wght` axis metadata (min/default/max) |
| `gvar` | Per-point outline deltas for the `wght` axis |
| `COLR` | Color glyph layers (v0) |
| `CPAL` | Color palette used by `COLR` |

## Generated features

| Tag | Lookup | Sample | Notes |
|---|---|---|---|
| `liga` | GSUB 4 | `fi` `fl` `ff` `ffi` `ffl` | Standard ligatures |
| `kern` | GPOS 2 + `kern` | `AV` `To` `Wa` | Pair positioning plus legacy fallback |
| `case` | GSUB 1 | `(A)` `[B]` | Raised punctuation for all-caps setting |
| `frac` | GSUB 4 | `1/2` `3/4` `1/8` | Precomposed fraction ligatures |
| `tnum` | GSUB 1 | `0123` | Tabular figures with equal advances |
| `ss01` | GSUB 1 | `a` | Alternate `a` |
| `smcp` | GSUB 1 | `Small Caps` | Uppercase scaled to x-height |
| `onum` | GSUB 1 | `0123456789` | Oldstyle figures with ascenders/descenders |
| `zero` | GSUB 1 | `0` | Slashed zero |
| `sups` | GSUB 1 | `x2` | Superscript figures |
| `subs` | GSUB 1 | `H2O` | Subscript figures |
| `ccmp` | GSUB 4 | `a` + `◌́` | Composes base+mark to an existing precomposed glyph |
| `mark` | GPOS 4 | `q́` | Top-center mark-to-base anchors |
| `mkmk` | GPOS 6 | `q́̄` | Stacks a second combining mark above the first |

Example:

```css
font-feature-settings: "liga" 1, "kern" 1, "frac" 1, "case" 1, "tnum" 1;
```

## Variable weight

Set `output.variable: true` to emit one real `wght` axis with `fvar` + `gvar`.
The axis range is the recipe weight ±200, clamped to 100–900; the recipe weight
is the default instance.

```css
font-variation-settings: "wght" 820;
```

Latin, Greek, Cyrillic, symbols, and generated feature glyphs interpolate.
CJK/Hangul/kana stay at their default outlines. Advances are unchanged because
`HVAR` is not emitted; the axis changes outlines, not metrics. Split CJK fonts
are static.

## Color glyph

Every generated font includes one COLR v0 / CPAL color glyph at `U+E000`:

- palette 0: the recipe accent color (BGRA)
- palette 1: white
- palette 2: black fallback

The glyph has an accent outer layer and a white inner layer. A black base
outline is kept as a fallback for software without COLR support. This is a
demonstration color glyph, not arbitrary color-font authoring.

```html
<span style="font-family: MyTypeface">&#xE000;</span>
```

## Verification

- `go test ./...` builds fonts and asserts the GSUB/GPOS/GDEF/`kern` tags and
  lookup types with fontTools when it is available.
- `scripts/validate-fonts.py` loads every glyph and the required core tables;
  the Go test is the one that asserts layout tables.
- HarfBuzz shaping was run manually for `1/2`, `(A)`, `0123`, `fi`, `a`, `abc`,
  `0123456789`, `0`, `123` (sup/sub), `q́`, `a` + `◌́` (`ccmp`), and
  `q́̄` (`mkmk`).
- fontTools asserts the `fvar` axis and non-zero `gvar` deltas for `A`; HarfBuzz
  glyph extents were checked at `wght` 420, 620, and 820.
- fontTools asserts the `COLR` base/layer records and the `CPAL` accent/white
  palette; the specimen screenshot was rendered in Chrome.
- `fontcreator learn <font> --json` reports the same tags and lookup types in
  external fonts, including variable axes and color tables when present.

## Deliberate limits

Generated fonts are static and Latin-focused. They do **not** include:

- additional variable axes beyond `wght`, or metric variations (`HVAR`/`MVAR`);
- arbitrary color glyph authoring beyond the `U+E000` COLR/CPAL demo, or SVG /
  `sbix` / CBDT color tables;
- contextual substitutions (`calt`, `cswh`) beyond the `ccmp` composition
  ligatures above;
- cursive attachment;
- Arabic, Hebrew, Indic, Thai, vertical layout, or complex shaping.

`fontcreator learn` still reports those structures when it finds them in an
external font, so the report is useful even when the generator cannot reproduce
them.
