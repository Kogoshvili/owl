import { Router, Route } from "preact-router"
import { route } from "preact-router"
import { HomePage } from "./pages/home"
import { FileDetailPage } from "./pages/file-detail"
import { SuggestionDetailPage } from "./pages/suggestion-detail"
import { ToastProvider } from "./hooks/toast"
import { ToastContainer } from "./components/toast"
import "./app.css"

export function App() {
  return (
    <ToastProvider>
      <div class="app">
        <header class="app-header">
          <div class="header-inner">
            <a href="/" class="logo" onClick={(e) => { e.preventDefault(); route("/") }}>Owl Folder Suggester</a>

          </div>
        </header>
        <main class="app-main">
          <Router>
            <Route path="/" component={HomePage} />
            <Route path="/files/:id" component={FileDetailPage} />
            <Route path="/suggestions/:id" component={SuggestionDetailPage} />
          </Router>
        </main>
        <ToastContainer />
      </div>
    </ToastProvider>
  )
}
