# fontcreator233 docs site

A local-first documentation site built with **Vite+** (`vp`), **React 19**, and
the **React Compiler**. It renders the repository's Markdown docs plus guide
pages for templates and BMFont export, with no CDN or runtime network calls —
Geist and Geist Mono are vendored locally from the specimen assets.

## Develop

```bash
cd docs-site
npm install
npm run dev      # vp dev
npm run build    # vp build -> dist/
npm run check    # vp check
```

`npm run build` produces a static `dist/` suitable for GitHub Pages. The
workflow in `.github/workflows/docs.yml` builds it on pushes to `main` and
deploys to Pages.

## Content

Pages are defined in `src/content.ts`. Most pages import the canonical Markdown
from `docs/` with Vite's `?raw` imports, so the site and the repository docs
never drift. Guide pages (start, templates, BMFont) live inline in `content.ts`.

Markdown links ending in `.md` are rewritten to in-site navigation; external
links open in a new tab.

## React Compiler

`vite.config.ts` enables the compiler through `@rolldown/plugin-babel` with
`babel-plugin-react-compiler`. You can verify it is active by searching the
built bundle for `react.memo_cache_sentinel`.
