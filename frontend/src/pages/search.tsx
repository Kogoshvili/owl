import { useState, useEffect, useRef } from "preact/hooks"
import { route } from "preact-router"
import { useSearch } from "../hooks/queries"
import { ALL_SCOPES, type SearchScope, type SearchFileResult } from "../api"

const SCOPE_LABELS: Record<SearchScope, string> = {
  filenames: "Filenames",
  content: "Content",
  comments: "Comments",
  tags: "Tags",
}

const SOURCE_COLORS: Record<string, string> = {
  filename: "match-filename",
  content: "match-content",
  comment: "match-comment",
  tag: "match-tag",
}

export function SearchPage() {
  const [query, setQuery] = useState("")
  const [debounced, setDebounced] = useState("")
  const [scopes, setScopes] = useState<Set<SearchScope>>(() => new Set(ALL_SCOPES))
  const inputRef = useRef<HTMLInputElement>(null)

  const activeScopes = Array.from(scopes) as SearchScope[]
  const searchQuery = useSearch(debounced, activeScopes)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const toggleScope = (scope: SearchScope) => {
    setScopes((prev) => {
      const next = new Set(prev)
      if (next.has(scope)) {
        if (next.size > 1) next.delete(scope)
      } else {
        next.add(scope)
      }
      return next
    })
  }

  const hasQuery = debounced.length >= 2
  const results = searchQuery.data
  const fileCount = results?.files?.length ?? 0

  return (
    <div class="page search-page">
      <div class="page-header">
        <h2>Search</h2>
      </div>

      <div class="search-bar">
        <input
          ref={inputRef}
          type="text"
          class="search-input"
          placeholder="Search files, tags, comments..."
          value={query}
          onInput={(e) => setQuery((e.target as HTMLInputElement).value)}
        />
        {query && (
          <button class="btn btn-sm search-clear" onClick={() => { setQuery(""); setDebounced("") }}>
            Clear
          </button>
        )}
      </div>

      <div class="scope-pills">
        {ALL_SCOPES.map((scope) => (
          <button
            key={scope}
            class={`scope-pill${scopes.has(scope) ? " active" : ""}`}
            onClick={() => toggleScope(scope)}
          >
            {SCOPE_LABELS[scope]}
          </button>
        ))}
      </div>

      {!hasQuery && (
        <div class="empty">Type at least 2 characters to search</div>
      )}

      {hasQuery && searchQuery.isLoading && (
        <div class="empty">Searching...</div>
      )}

      {hasQuery && searchQuery.isError && (
        <div class="empty">Search failed: {searchQuery.error?.message}</div>
      )}

      {hasQuery && !searchQuery.isLoading && results && (
        <>
          <div class="search-summary">
            {fileCount === 0
              ? "No results found"
              : `${fileCount} result${fileCount !== 1 ? "s" : ""}`}
          </div>

          {fileCount > 0 && (
            <div class="search-section">
              <h3>Files</h3>
              <div class="search-results">
                {results.files.map((r: SearchFileResult) => (
                  <div class="search-result" key={r.file_id} onClick={() => route(`/files/${r.file_id}`)} style="cursor:pointer">
                    <div class="search-result-top">
                      <span class="search-result-name">{r.name}</span>
                      <div class="search-result-badges">
                        {r.match_sources.map((s) => (
                          <span key={s} class={`match-badge ${SOURCE_COLORS[s] ?? ""}`}>{s}</span>
                        ))}
                      </div>
                    </div>
                    <div class="search-result-path" title={r.path}>{r.path}</div>
                    {r.snippet && (
                      <div class="search-result-snippet">{r.snippet}</div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
