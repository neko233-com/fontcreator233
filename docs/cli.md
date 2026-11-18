# CLI reference

```text
fontcreator <command> [flags] [arguments]
```

Global behavior:

- TUI when stderr is a TTY; `--plain`, `--no-tui`, `--json`, `CI=true`, or a
  pipe falls back to line/JSON output.
- `--json` writes exactly one envelope to stdout. Logs and progress go to stderr.
- Exit codes: `0` success, `1` user/recipe error, `2` internal error,
  `130` canceled (Ctrl+C/SIGTERM).
- Generated files are workspace-tracked under `<out>/.fontcreator233/`.

## build

```bash
fontcreator build recipe.yaml --out dist --target all --jobs 8 --no-open
```

Flags: `--out`, `--target` (repeatable/comma), `--split-cjk`, `--jobs`,
`--assets-dir`, `--self-contained-html`, `--no-auto-clean`, `--force`,
`--no-open`, `--plain`, `--no-tui`, `--json`, `--quiet`.

## build-all

```bash
fontcreator build-all --dir examples --out dist --font-jobs 3 --jobs 4
```

Builds every `.yaml`/`.yml`/`.json` recipe in a directory concurrently with a
single workspace lock and commit. Each font gets its own namespace, so
packages and manifests never overwrite each other:

```text
dist/
├─ AuroraGrotesk-Regular/
│  ├─ AuroraGrotesk-Regular.ttf
│  ├─ web/ system/ unity/ unreal/
│  └─ fontcreator233.manifest.json
└─ NoirDidone-NoirItalic/
   └─ ...
```

Duplicate family/style stems are rejected before building.

## validate / inspect / specimen

```bash
fontcreator validate recipe.yaml --json
fontcreator inspect dist/Font-Regular.ttf --json
fontcreator specimen dist/Font-Regular.ttf --out dist --open
```

## init / describe / schema

```bash
fontcreator init --dir work --template geometric|serif|display --family "Name"
fontcreator describe --json
fontcreator describe --markdown
fontcreator schema
```

## library

```bash
fontcreator library list --json
fontcreator library export --dir glyphs --recipe recipe.yaml
```

## workspace

```bash
fontcreator status --out dist --json
fontcreator clean --out dist --dry-run          # staging/backup scratch only
fontcreator clean --out dist --all --dry-run    # preview managed outputs
fontcreator clean --out dist --all --yes
fontcreator clean --out dist --target unity --yes
fontcreator clean --out dist --older-than 7d --yes
```

`clean` never deletes untracked files or managed files whose SHA-256 changed,
unless `--force`. Deletion requires `--yes` unless `--dry-run`. A plain
`clean` is safe maintenance: it only removes staging/backup scratch and prunes
old build history; use `--all` for generated outputs.

## import / export

```bash
fontcreator import font MyFont.ttf --out work/import --extract-glyphs --json
fontcreator import font MyFont.ttf --out work/import --force --json
fontcreator import bundle dist/Font.bundle.zip --out work/restored --build --json
fontcreator export glyphs dist/Font.ttf --dir glyphs --unicodes 0041-007A --json
fontcreator export glyphs --recipe recipe.yaml --dir glyphs --scripts latin --json
fontcreator export bundle recipe.yaml --out Font.bundle.zip --target all --json
```

`import font` writes `profile.json`, a starter `recipe.yaml`, and optional SVG
glyphs. `import bundle` validates the zip, checksums, paths, and OFL file before
extracting.

## mcp

```bash
fontcreator mcp
```

Newline-delimited JSON-RPC 2.0 over stdio. Tools:

- `fontcreator_describe`
- `fontcreator_validate`
- `fontcreator_build`
- `fontcreator_inspect`
- `fontcreator_specimen`
- `fontcreator_library_list`
- `fontcreator_status`

Build calls default to `open: false`.

## version

```bash
fontcreator version --json
```
