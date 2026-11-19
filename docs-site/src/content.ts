import cli from '../../docs/cli.md?raw'
import opentype from '../../docs/opentype.md?raw'
import platforms from '../../docs/platforms.md?raw'
import specimen from '../../docs/web-specimen.md?raw'
import workflow from '../../docs/agent-workflow.md?raw'

export type Page = {
  id: string
  title: string
  group: string
  markdown: string
}

const start = `# fontcreator233

Turn a reference image and a reviewable \`recipe.yaml\` into a real, installable
font — fully offline and deterministic. You supply the visual judgment; the CLI
writes TTF/WOFF fonts, a local specimen, and integration packages.

## Install

\`\`\`bash
go install github.com/neko233-com/fontcreator233@latest
fontcreator describe --json
\`\`\`

## First font

\`\`\`bash
fontcreator init --dir work --template geometric --family "My Typeface"
fontcreator validate work/font.recipe.yaml --json
fontcreator build work/font.recipe.yaml --out dist --target all --json --no-open
fontcreator specimen dist/MyTypeface-Regular.ttf --open
\`\`\`

## Iterate

Keep the \`seed\` fixed so outline changes come from your parameter edits, not
noise. Work in three passes: structure first, texture second, terminals last.

\`\`\`bash
fontcreator learn path/to/reference.ttf --out work/learned --json
\`\`\`

\`learn\` profiles external TTF/OTF files — names, metrics, cmap, GSUB/GPOS
features, kerning, variable axes, and color tables — and writes a starter
recipe. It never copies or redistributes the reference font.
`

const templates = `# Reference templates

Five ready-to-run recipes live in \`templates/\`. Each is valid out of the box
and meant to be copied into your own \`work/\` directory and edited.

| Template | Use | Notable settings |
|---|---|---|
| \`code-editor\` | Editors, terminals, code review | high x-height, even rhythm, \`calt\` arrows, BMFont output |
| \`product-ui\` | Dashboards, navigation, product UI | humanist skeleton, restrained contrast |
| \`poster-display\` | Headlines, posters, social graphics | wide 118%, weight 800, flared terminals |
| \`editorial-serif\` | Long-form articles, books | bracketed serif, contrast 0.76 |
| \`cjk-partner\` | Bilingual Latin + CJK products | CJK base coverage, \`splitCJK: 4096\` |

\`\`\`bash
fontcreator validate templates/code-editor.recipe.yaml --json
fontcreator build templates/code-editor.recipe.yaml --out dist --target all
\`\`\`

## Domain styles

The templates encode design traits, not copied fonts. The code-editor preset
borrows the *ideas* behind programming typefaces — a tall x-height, open
apertures, moderate weight, tight contrast range, and a stable rhythm — without
embedding any external font. Use \`style.tracking\` to tune spacing: positive
values open text up for small sizes, negative values tighten display setting.

The specimen has its own live tracking slider, so you can preview letter-spacing
changes before writing them into the recipe.
`

const bmfont = `# BMFont export

\`--target bmfont\` rasterizes the generated outlines into an AngelCode BMFont
v3 \`.fnt\` plus PNG atlas pages. Everything is generated locally.

\`\`\`yaml
output:
  targets: [universal, bmfont]
  bmfont:
    size: 32
    padding: 2
    atlasSize: 512
    charset: latin
\`\`\`

| Option | Meaning |
|---|---|
| \`size\` | Pixel size of the em square (8–256) |
| \`padding\` | Per-glyph atlas padding (0–16) |
| \`atlasSize\` | Square atlas page size (64–4096) |
| \`charset\` | \`latin\`, \`latin-ext\`, \`all\`, or an explicit rune string |

Glyphs are rasterized with 4× supersampling and an even-odd scanline fill,
cropped tightly, then shelf-packed into one or more atlas pages. The \`.fnt\`
contains info, common, pages, chars, and kerning blocks.

Because BMFont is a bitmap format, pick the charset deliberately. \`latin\`
emits U+0020–U+007E; \`all\` includes every cmap glyph and can produce many pages.
`

export const pages: Page[] = [
  { id: 'start', title: 'Getting started', group: 'Guide', markdown: start },
  { id: 'templates', title: 'Reference templates', group: 'Guide', markdown: templates },
  { id: 'bmfont', title: 'BMFont export', group: 'Guide', markdown: bmfont },
  { id: 'cli', title: 'CLI reference', group: 'Reference', markdown: cli },
  { id: 'opentype', title: 'OpenType features', group: 'Reference', markdown: opentype },
  { id: 'platforms', title: 'Platform packages', group: 'Reference', markdown: platforms },
  { id: 'specimen', title: 'Web specimen', group: 'Reference', markdown: specimen },
  { id: 'workflow', title: 'Agent workflow', group: 'Reference', markdown: workflow },
]
