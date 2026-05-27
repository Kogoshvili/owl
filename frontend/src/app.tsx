import { Router, Route } from "preact-router"
import { route } from "preact-router"
import { DashboardPage } from "./pages/dashboard"
import { FilesPage } from "./pages/files"
import { WatchedDirsPage } from "./pages/watched-dirs"
import { NotesPage } from "./pages/notes"
import { SearchPage } from "./pages/search"
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
            <NavLink href="/">Dashboard</NavLink>
            <NavLink href="/files">Files</NavLink>
            <NavLink href="/watched-directories">Directories</NavLink>
            <NavLink href="/notes">Notes</NavLink>
            <NavLink href="/search">Search</NavLink>
          </nav>
        </div>
      </header>
      <main class="app-main">
        <Router>
          <Route path="/" component={DashboardPage} />
          <Route path="/files" component={FilesPage} />
          <Route path="/watched-directories" component={WatchedDirsPage} />
          <Route path="/notes" component={NotesPage} />
          <Route path="/search" component={SearchPage} />
        </Router>
      </main>
    </div>
  )
}
