# OpenType features

fontcreator233 writes real OpenType layout tables into every generated TTF/WOFF.
The specimen exposes the features live; HarfBuzz and fontTools verify them in
tests.

## Generated tables

| Table | Purpose |
|---|---|
| `GSUB` | Ligatures and one-for-one substitutions |
| `GPOS` | Pair kerning and mark-to-base attachment |
| `GDEF` | Glyph classes for marks and alternates |
| `kern` | Legacy pair kerning for older software |

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
| `mark` | GPOS 4 | `q́` | Top-center mark-to-base anchors |

Example:

```css
font-feature-settings: "liga" 1, "kern" 1, "frac" 1, "case" 1, "tnum" 1;
```

## Verification

- `go test ./...` builds fonts and asserts the GSUB/GPOS/GDEF/`kern` tags and
  lookup types with fontTools when it is available.
- `scripts/validate-fonts.py` loads every glyph and the required core tables;
  the Go test is the one that asserts layout tables.
- HarfBuzz shaping was run manually for `1/2`, `(A)`, `0123`, `fi`, `a`, `abc`,
  `0123456789`, `0`, `123` (sup/sub), and `q́`.
- `fontcreator learn <font> --json` reports the same tags and lookup types in
  external fonts, including variable axes and color tables when present.

## Deliberate limits

Generated fonts are static and Latin-focused. They do **not** include:

- variable axes (`fvar`/`gvar`/`HVAR`) or color layers (`COLR`/`CPAL`/`SVG `);
- contextual substitutions (`calt`, `cswh`, complex `ccmp`);
- mark-to-mark attachment (`mkmk`) or cursive attachment;
- Arabic, Hebrew, Indic, Thai, vertical layout, or complex shaping.

`fontcreator learn` still reports those structures when it finds them in an
external font, so the report is useful even when the generator cannot reproduce
them.
