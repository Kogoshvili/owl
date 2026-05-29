import { Router, Route } from "preact-router"
import { route } from "preact-router"
import { IngestPage } from "./pages/ingest"
import { FilesPage } from "./pages/files"
import { SearchPage } from "./pages/search"
import { FileDetailPage } from "./pages/file-detail"
import { SuggestionsPage } from "./pages/suggestions"
import { SuggestionDetailPage } from "./pages/suggestion-detail"
import "./app.css"

function NavLink({ href, children }: { href: string; children: string }) {
  const handleClick = (e: MouseEvent) => {
    e.preventDefault()
    route(href)
  }
  return (
    <a href={href} class="nav-link" onClick={handleClick}>
      {children}
    </a>
  )
}

export function App() {
  return (
    <div class="app">
      <header class="app-header">
        <div class="header-inner">
          <a href="/" class="logo" onClick={(e) => { e.preventDefault(); route("/") }}>Owl File Manager</a>
          <nav class="nav">
            <NavLink href="/">Ingest</NavLink>
            <NavLink href="/files">Files</NavLink>
            <NavLink href="/search">Search</NavLink>
            <NavLink href="/suggestions">Suggestions</NavLink>
          </nav>
        </div>
      </header>
      <main class="app-main">
        <Router>
          <Route path="/" component={IngestPage} />
          <Route path="/files" component={FilesPage} />
          <Route path="/files/:id" component={FileDetailPage} />
          <Route path="/search" component={SearchPage} />
          <Route path="/suggestions" component={SuggestionsPage} />
          <Route path="/suggestions/:id" component={SuggestionDetailPage} />
        </Router>
      </main>
    </div>
  )
}
