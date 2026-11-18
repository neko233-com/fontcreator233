---
name: fontcreator233
description: Create deterministic multilingual fonts from reference images with the fontcreator233 CLI. Use when the user wants a font matching concept art, a poster, a brand, or a moodboard; needs TTF/WOFF files, a local HTML specimen, glyph libraries, or Unity TMP / Unreal / Web / System packages.
---

# fontcreator233

The agent owns visual understanding. fontcreator233 is the deterministic,
offline backend that turns a recipe into a real font.

## Workflow

1. Read the reference image yourself. Write a short style brief: weight,
   contrast, serif, width, terminals, slant, mood, motion.
2. Call `fontcreator describe --json` to read the current contract and schema.
   If the user supplied an existing TTF/OTF as a reference, run
   `fontcreator learn path/to/reference.ttf --out work/learned --json` and read
   its metrics, features, kerning, axes, and starter recipe. Treat it as data,
   not as permission to copy a proprietary font.
3. Run `fontcreator init --dir work --template geometric|serif|display`.
4. Edit `work/font.recipe.yaml`. Use the mapping below. Keep `seed` fixed
   while iterating.
5. `fontcreator validate work/font.recipe.yaml --json`.
6. `fontcreator build work/font.recipe.yaml --out dist --target all --json --no-open`.
7. `fontcreator specimen dist/<font>.ttf --open` for human review.
8. `fontcreator inspect dist/<font>.ttf --json`; read `coverage`,
   `coverageReport`, and `warnings`. Never claim unsupported coverage.
9. Iterate at most three times: proportions → contrast/details → terminals.
10. Report the recipe, font, specimen, platform packages, SHA-256, and known
    limits.

## Visual mapping

| Reference signal | Recipe |
|---|---|
| hairline / thin strokes | `style.weight` 200–350 |
| regular text weight | `style.weight` 400–520 |
| bold / poster weight | `style.weight` 620–800 |
| uniform stroke width | `style.contrast` 0–0.1 |
| high contrast / Didone | `style.contrast` 0.7–0.95, `style.serif: hairline` |
| geometric sans | `style.skeleton: geometric`, `style.serif: none` |
| humanist sans | `style.skeleton: humanist`, `style.contrast: 0.25–0.45` |
| slab serif | `style.serif: slab`, `serifSize: 0.5–0.8` |
| rounded / soft | `style.terminal: round` |
| cut / sharp | `style.terminal: angled` |
| Art Nouveau / organic | `style.terminal: flare`, `contrast: 0.4–0.7`, `roughness: 0.1–0.3` |
| condensed poster | `style.width: 82–92` |
| wide display | `style.width: 108–122` |
| italic reference | `style.slant: 6–12` |

Always use the agent's own vision. Do not ask the CLI to analyze an image.

## Package targets

- `--target all` writes `universal`, `web`, `system`, `unity`, `unreal`, `apps`.
- Apps: `apps/` contains per-app guides for Photoshop, Illustrator, InDesign,
  剪映/CapCut, Premiere, After Effects, DaVinci, Final Cut, Figma, Canva,
  Office, OBS, Blender, Godot, Unity, and Unreal, plus SRT/ASS caption samples.
  Tell the user which apps read system fonts and which load the TTF directly.
- Unity: `unity/` is a UPM package. `Editor/FontCreator233TMPImporter.cs`
  creates TMP Font Assets, multi-atlas settings, charsets, and fallback chains.
  The generator does not run Unity. Tell the user to install TMP and use
  `Tools/Fontcreator233/...`.
- Unreal: `unreal/Scripts/fontcreator233_import.py` imports Font assets and
  prints CompositeFont ranges when the UE Python API cannot create one.
- Web: `web/font.css` plus local font files; split fonts use `unicode-range`.
- System: `system/INSTALL.md`.

## Glyph library and imports

- `fontcreator library list --json`
- `fontcreator library export --dir glyphs --recipe work/font.recipe.yaml`
- Override glyphs with `glyphs.overrides: glyphs` and files named
  `U+4E2D.svg`. Coordinates are font units, Y up, baseline 0,
  `viewBox="0 -250 1000 1250"`, with `data-advance="600"`.
- `fontcreator learn <font> --out work/learned --json` for name records,
  metrics, cmap coverage, GSUB/GPOS features, kerning, axes, and a starter recipe.
- `fontcreator import font <font> --out work/import --extract-glyphs`
- `fontcreator export bundle` / `import bundle` for shareable zip bundles.
- `fontcreator export glyphs <font> --dir out --unicodes 0041-007A`

## Guardrails

- Do not claim Arabic, Hebrew, Indic, Thai, vertical layout, or advanced shaping.
- `coverage.base.cjk` = actual embedded cmap coverage, not every Unicode block.
- Fonts using embedded base glyphs must be OFL-1.1; never change the license.
- Never use `Noto` as a family name.
- Generated output is git-ignored (`dist/`, `work/`); do not commit it.
- Use `--json` / `--plain` / `--no-tui` in automation; TUI is for humans.
- Specimens are local/offline. Do not add CDNs or external scripts.
