import { useEffect, useMemo, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { pages } from './content'

function pageFromHash(): string {
  const id = window.location.hash.replace(/^#\/?/, '')
  return pages.some((p) => p.id === id) ? id : pages[0].id
}

function targetId(href: string | undefined): string | null {
  if (!href) return null
  const clean = href.split('#')[0].replace(/^\.\//, '')
  const base = clean.split('/').pop() ?? ''
  if (!base.endsWith('.md')) return null
  const id = base.replace(/\.md$/, '')
  if (id === 'README') return 'start'
  return pages.some((p) => p.id === id) ? id : null
}

export default function App() {
  const [active, setActive] = useState(pageFromHash)
  const [dark, setDark] = useState(() => {
    const saved = localStorage.getItem('fc233-theme')
    if (saved) return saved === 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    const onHash = () => setActive(pageFromHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  useEffect(() => {
    document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    localStorage.setItem('fc233-theme', dark ? 'dark' : 'light')
  }, [dark])

  const page = useMemo(() => pages.find((p) => p.id === active) ?? pages[0], [active])
  const groups = useMemo(() => {
    const out = new Map<string, typeof pages>()
    for (const p of pages) {
      const list = out.get(p.group) ?? []
      list.push(p)
      out.set(p.group, list)
    }
    return out
  }, [])

  const navigate = (id: string) => {
    window.location.hash = `/${id}`
    setActive(id)
    window.scrollTo({ top: 0 })
  }

  return (
    <div className="app">
      <header className="topbar">
        <button className="brand" onClick={() => navigate('start')}>
          <span className="brand-mark" aria-hidden="true" />
          fontcreator<span>233</span>
        </button>
        <nav className="topnav">
          <a href="https://github.com/neko233-com/fontcreator233" target="_blank" rel="noreferrer">
            GitHub
          </a>
          <a href="https://github.com/neko233-com/fontcreator233/releases" target="_blank" rel="noreferrer">
            Releases
          </a>
          <button className="icon" onClick={() => setDark((v) => !v)} aria-label="Toggle theme">
            {dark ? '☀' : '◐'}
          </button>
        </nav>
      </header>

      <div className="layout">
        <aside className="sidebar">
          {[...groups.entries()].map(([group, items]) => (
            <div className="group" key={group}>
              <div className="group-label">{group}</div>
              {items.map((item) => (
                <button
                  key={item.id}
                  className={`nav-item ${item.id === active ? 'active' : ''}`}
                  onClick={() => navigate(item.id)}
                >
                  {item.title}
                </button>
              ))}
            </div>
          ))}
        </aside>

        <main className="content">
          <div className="eyebrow">{page.group}</div>
          <article className="markdown">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                a({ href, children, ...props }) {
                  const id = targetId(href)
                  if (id) {
                    return (
                      <button className="inline-link" onClick={() => navigate(id)}>
                        {children}
                      </button>
                    )
                  }
                  return (
                    <a href={href} target="_blank" rel="noreferrer" {...props}>
                      {children}
                    </a>
                  )
                },
              }}
            >
              {page.markdown}
            </ReactMarkdown>
          </article>
          <footer className="footer">
            fontcreator233 · deterministic, offline font creation · MIT / OFL-1.1
          </footer>
        </main>
      </div>
    </div>
  )
}
