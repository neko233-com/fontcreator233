# fontcreator233 — agent guide

fontcreator233 is a deterministic, offline font creator. **You** (the agent) read
the reference image and decide the style. The CLI never calls a vision model.
It turns a reviewable `recipe.yaml` into real TTF/WOFF fonts, a local HTML
specimen, Unity/Unreal/Web/System packages, and an app handoff pack for
Photoshop, 剪映/CapCut, Premiere, Figma, Office, and other editors.

## First call

```bash
fontcreator describe --json
```

That returns the command list, recipe JSON Schema, style mapping, coverage
options, platform notes, and limits. `fontcreator describe --markdown` is the
same contract as a readable guide.

## Recommended loop

```bash
fontcreator init --dir work --template geometric --family "My Typeface"
# edit work/font.recipe.yaml using the style mapping
fontcreator validate work/font.recipe.yaml --json
fontcreator build work/font.recipe.yaml --out dist --target all --json --no-open
fontcreator specimen dist/MyTypeface-Regular.ttf --open
fontcreator inspect dist/MyTypeface-Regular.ttf --json
fontcreator learn path/to/reference.ttf --out work/learned --json
fontcreator status --out dist --json
```

`output.variable: true` emits one real `wght` axis via `fvar` + `gvar` plus
`HVAR` advance and `MVAR` metric deltas. The `init` template and examples enable
it; set it to `false` for a static font. The axis varies Latin/Greek/Cyrillic/
symbol and generated feature glyphs; CJK, Hangul, and kana stay static.
`openType.color` emits recipe-driven multi-glyph COLR v0 with multiple CPAL
palettes; without it one COLR/CPAL accent glyph is emitted at `U+E000` using
`meta.accent`. `calt` turns the literal `->` into an arrow
through a chain-context lookup; the specimen has a toggle for it. Add further
exact-sequence rules under `openType.contextual`. `output.cursive: true` emits a
real GPOS `curs` cursive attachment lookup (type 3 entry/exit anchors on `a`–`z`);
the specimen has a toggle for it too.

Iterate at most three times: structure first, then texture, then terminals.

## What to decide from the reference

| Visual signal | Recipe |
|---|---|
| thin / regular / bold strokes | `style.weight` 200–350 / 400–520 / 620–800 |
| uniform vs high contrast | `style.contrast` 0–.1 vs .7–.95 |
| slab / Didone / humanist serif | `style.serif` slab / hairline / bracketed |
| rounded / cut / flared terminals | `style.terminal` round / angled / flare |
| condensed / wide display | `style.width` 82–92 / 108–122 |
| italic reference | `style.slant` 6–12 |
| hand-drawn or organic | `style.roughness` .1–.3 + fixed `seed` |

Keep the seed fixed during iteration so outline changes are attributable to
your parameter changes, not noise.

## Agent-native contract

- `--json` prints exactly one JSON envelope on stdout; logs go to stderr.
- Exit codes: 0 success, 1 recipe/user error, 2 internal, 130 canceled.
- `fontcreator mcp` runs a stdio MCP server with:
  `fontcreator_describe`, `fontcreator_validate`, `fontcreator_build`,
  `fontcreator_inspect`, `fontcreator_learn`, `fontcreator_specimen`,
  `fontcreator_library_list`, `fontcreator_status`.
- TUI is for humans; use `--json`, `--plain`, or `--no-tui` in automation.
- Builds are byte-identical within the same tool version, OS/arch, and Go
  version. `--jobs 1` and `--jobs 8` match on the same platform.

## Glyph library

- `fontcreator library list --json` / `library export --dir glyphs`.
- Override individual glyphs with `glyphs.overrides` pointing at
  `U+XXXX.svg` files. SVG coordinates are font units, Y up, baseline 0,
  `viewBox="0 -250 1000 1250"`, with `data-advance="600"`.
- Use `fontcreator import font <font.ttf> --out work/imported --extract-glyphs`
  to profile a font and export SVG glyphs.
- Bundle sharing: `fontcreator export bundle recipe --out font.zip` and
  `fontcreator import bundle font.zip --out work/restored`.

## Guardrails

- Do not claim Arabic, Hebrew, Indic, Thai, vertical layout, or complex
  shaping support.
- `coverage.base.cjk` means **actual embedded cmap coverage**, not all Unicode.
  Read `coverageReport` and `warnings`.
- Any output using embedded base glyphs is OFL-1.1. The recipe cannot override
  this.
- Do not use `Noto` as a family name.
- Unity/UE packages are generated, not tested against their Editors.
- Generated output (`dist/`, `work/`, `.fontcreator233/`) is git-ignored.
- Specimen assets are local; never add a CDN to the HTML.
