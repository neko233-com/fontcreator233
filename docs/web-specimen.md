# Offline HTML specimen

The specimen is a real review surface, not a debug dump. It is designed in the
Vercel/Geist visual language and works directly from `file://` with no network.

## Layout

- Sticky monochrome navigation with font-load status.
- Hero with a giant `Aa` drawn by the generated font, metadata, and CTAs.
- Ticker across Latin, CJK, Greek, Cyrillic, and Hangul.
- Type tester: editable headline with size, leading, and tracking controls.
- OpenType playground: `liga`, `kern`, `frac`, `case`, `tnum`, and `ss01`
  toggles, plus a feature grid for `smcp`, `onum`, `zero`, `sups`, `subs`, and
  `mark`; live `font-feature-settings` and feature-table metadata.
- Scale waterfall from caption to poster.
- Global coverage cards for Latin, Greek, Russian, Chinese, Japanese, Korean,
  and symbols.
- App handoff section: Photoshop, 剪映/CapCut, Premiere, After Effects, DaVinci,
  Figma, Office, Unity/Unreal; caption preview and poster preview.
- Virtualized glyph library with character/codepoint search and script filter.
- Webfont CSS, details, license note, and a closing panel.

## Assets

```text
assets/
├─ specimen.css
├─ specimen.js
├─ vendor/tanstack-virtual.iife.js
├─ geist/Geist-Variable.woff2
├─ geist/GeistMono-Variable.woff2
├─ licenses/Geist-OFL.txt
├─ licenses/TanStack-Virtual-MIT.txt
└─ versions.json
```

- All references are relative; there are no CDN URLs or ES modules.
- Geist Sans/Mono provide the UI voice; the generated font is used for all
  specimen content.
- `@tanstack/virtual-core` (MIT) is bundled to a classic script for `file://`
  compatibility; only visible rows are rendered.
- `versions.json` records asset hashes; `workspace` reuses unchanged files.

## Dark/light

The page follows `prefers-color-scheme` and can be toggled in the nav. The accent
comes from `meta.accent` and is applied as a CSS variable.

## Self-contained option

`fontcreator build ... --self-contained-html` inlines CSS, JS, TanStack Virtual,
Geist WOFF2, glyph data, and the generated font as base64. It is larger but
single-file; the default external-assets layout stays easier to share and cache.

## Regeneration

```bash
fontcreator specimen dist/Font-Regular.ttf --out dist
```

Assets are embedded in the Go binary and copied next to the HTML. The glyph grid
data is written as `Font-Regular.glyphs.js` (`window.FontCreatorData`), so the
page never needs `fetch()` on `file://`.

## Validation

- Go tests assert local asset references, no external `src`/`href`, no
  `type="module"`, and that glyph data is present.
- Chrome headless was used during development to visually verify dark/light
  rendering, font loading, and the glyph grid.
- `document.fonts.check('16px "Specimen"')` drives the status pill and error
  banner; a missing font shows a clear message instead of a blank page.
- A missing `*.glyphs.js` or `assets/` directory raises a visible asset banner;
  the page never silently renders an empty grid.
