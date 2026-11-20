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

`--target bmfont` (included in `all`) writes an AngelCode BMFont `.fnt` and PNG
atlas pages from the generated outlines; configure size/padding/atlasSize/charset
under `output.bmfont`.

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

## Release & distribution

### Supported platforms

| Platform | Arch | Artifact |
|---|---|---|
| Windows | amd64 | `fontcreator233-vX.Y.Z-windows-amd64.tar.gz` + `.msi` (WiX 5.0.2) |
| macOS | amd64 | `fontcreator233-vX.Y.Z-darwin-amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `fontcreator233-vX.Y.Z-darwin-arm64.tar.gz` |
| Linux | amd64 | `fontcreator233-vX.Y.Z-linux-amd64.tar.gz` |

Artifact suffixes come from Go's `GOOS-GOARCH` (`darwin`, not `macos`) so they match
`runtime.GOOS` in `fontcreator update`.

**Windows is the only platform that ships an MSI.** macOS/Linux ship portable tarballs.
Three architectures is the full matrix — do **not** add Linux arm64, 32-bit, or
BSD/Solaris builds unless explicitly asked.

### Version injection

The binary version is set via `-ldflags` at release-build time:

```bash
go build -ldflags "-s -w \
  -X github.com/neko233-com/fontcreator233/internal/agentschema.Version=v1.0.0 \
  -X github.com/neko233-com/fontcreator233/internal/agentschema.Commit=$(git rev-parse --short HEAD) \
  -X github.com/neko233-com/fontcreator233/internal/agentschema.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -trimpath -o fontcreator .
```

`fontcreator version --json` emits the full BuildInfo:
`version`, `commit`, `buildDate`, `goVersion`, `os`, `arch`.

### Release — local only, no GitHub Actions

**There is no release workflow.** `.github/workflows/` must not contain anything that
publishes releases; publishing is done from a developer machine so the artifacts are
built and inspected before they go public. `ci.yml` only runs format/vet/test.

One command does everything — cross-compile 3 platforms, build the MSI, create and
push a new tag, publish the release:

```powershell
.\scripts\release-local.ps1 -Version v1.0.0
.\scripts\release-local.ps1 -Version v1.0.0 -SkipTest   # skip go test
.\scripts\release-local.ps1 -Version v1.0.0 -NoUpload   # package + tag only
.\scripts\release-local.ps1 -Version v1.0.0 -NoTag      # rebuild assets only
```

It refuses to run on a dirty working tree or an existing tag, and only uploads after
`gofmt`, `go vet`, and (unless `-SkipTest`) `go test ./...` pass.

Produced and uploaded assets:

```
fontcreator233-vX.Y.Z-windows-amd64.tar.gz  (+ .sha256)
fontcreator233-vX.Y.Z-windows-amd64.msi     (+ .sha256)
fontcreator233-vX.Y.Z-darwin-amd64.tar.gz   (+ .sha256)
fontcreator233-vX.Y.Z-darwin-arm64.tar.gz   (+ .sha256)
```

A Windows MSI is built with **WiX 5.0.2** (`dotnet tool install --global wix --version 5.0.2`)
from `scripts/installer/fontcreator.wxs` (schema v4 `<Package>`, NOT `<Product>`).
Do **not** bump to WiX 6/7: those require the paid OSMF EULA. `build-msi.ps1`
auto-pins 5.0.2 under `%TEMP%` when it detects a v6/v7 install.

### Quick local bake (Windows only, no tag/upload)

```powershell
.\scripts\build-release.ps1 -Version v1.0.0            # tar.gz only
.\scripts\build-release.ps1 -Version v1.0.0 -WithMSI   # tar.gz + msi
```

Use this for a fast sanity build; use `release-local.ps1` to actually publish.

### Update flow

Users update with **no external dependency** — the Go binary downloads and
replaces itself:

```bash
fontcreator update --check   # only show if newer available
fontcreator update           # download + verify SHA256 + replace
```

The update client tries these mirrors **in order**; first reachable wins:
1. `https://github.com` (primary)
2. `https://ghproxy.net/https://github.com`
3. `https://mirror.ghproxy.com/https://github.com`
4. `https://ghfast.top/https://github.com`

Each mirror also serves the `.sha256` companion file so the client verifies
checksum before replacing.

### One-click installer scripts

Windows (portable zip, no admin needed):
```powershell
powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/neko233-com/fontcreator233/main/scripts/install.ps1 | iex"
```
With MSI (recommended, needs admin):
```powershell
powershell -File install.ps1 -UseMSI
```

macOS / Linux:
```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/fontcreator233/main/scripts/install.sh | bash
```

Both scripts auto-detect latest tag, probe mirrors, verify SHA256, extract
to `$HOME/.local/bin` (or custom path), and add the install dir to PATH.

### What to release and what not

**DO release:** binary artifacts, `.sha256` checksums, MSI on Windows.
**DO NOT release:** test fixtures, `dist/`, `work/`, `.fontcreator233/`, the
`assets/` source module, Go build cache, `.exe.new` leftover from partial
updates.

### MSI design

- `<Package>` (WiX v4+) with `Id="*"` (auto-generate), `UpgradeCode` fixed
  across all releases so `MajorUpgrade` replaces old versions cleanly.
- Installs to `%ProgramFiles%\fontcreator233\` as `perMachine`.
- Adds the install dir to HKLM PATH via `Environment` element.
- **No Start Menu shortcut** — this is a CLI tool, not a GUI app.
- **No auto-update registry entry** — update is manual via `fontcreator update`.

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
- `docs/assets/*.png` are README screenshots of the specimen. When the specimen CSS,
  JS, or the generated metrics change, regenerate them from a fresh build in dark
  mode at 1760×1100 before releasing — stale screenshots are worse than none.
