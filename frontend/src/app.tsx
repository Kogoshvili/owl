import { Router, Route } from "preact-router"
import { route } from "preact-router"
import { HomePage } from "./pages/home"
import { FileDetailPage } from "./pages/file-detail"
import { SuggestionsPage } from "./pages/suggestions"
import { SuggestionDetailPage } from "./pages/suggestion-detail"
import { ToastProvider } from "./hooks/toast"
import { ToastContainer } from "./components/toast"
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
    <ToastProvider>
      <div class="app">
        <header class="app-header">
          <div class="header-inner">
            <a href="/" class="logo" onClick={(e) => { e.preventDefault(); route("/") }}>Owl File Manager</a>
            <nav class="nav">
              <NavLink href="/">Files</NavLink>
              <NavLink href="/suggestions">Suggestions</NavLink>
            </nav>
          </div>
        </header>
        <main class="app-main">
          <Router>
            <Route path="/" component={HomePage} />
            <Route path="/files/:id" component={FileDetailPage} />
            <Route path="/suggestions" component={SuggestionsPage} />
            <Route path="/suggestions/:id" component={SuggestionDetailPage} />
          </Router>
        </main>
        <ToastContainer />
      </div>
    </ToastProvider>
  )
}
