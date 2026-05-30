import { Router, Route } from "preact-router"
import { route } from "preact-router"
import { useQueryClient } from "@tanstack/preact-query"
import { HomePage } from "./pages/home"
import { FileDetailPage } from "./pages/file-detail"
import { SuggestionDetailPage } from "./pages/suggestion-detail"
import { ToastProvider } from "./hooks/toast"
import { ToastContainer } from "./components/toast"
import "./app.css"

export function App() {
  const qc = useQueryClient()

  const handleRefresh = () => {
    qc.invalidateQueries({ queryKey: ["physicalFolders"] })
    qc.invalidateQueries({ queryKey: ["folderGuards"] })
    qc.invalidateQueries({ queryKey: ["processingStats"] })
    qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
    qc.invalidateQueries({ queryKey: ["suggestions"] })
    qc.invalidateQueries({ queryKey: ["watchedDirs"] })
    qc.invalidateQueries({ queryKey: ["llmStatus"] })
  }

  return (
    <ToastProvider>
      <div class="app">
        <header class="app-header">
          <div class="header-inner">
            <a href="/" class="logo" onClick={(e) => { e.preventDefault(); route("/") }}>Owl Folder Suggester</a>
            <button class="btn btn-sm" style="margin-left:auto" onClick={handleRefresh}>Refresh</button>
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
