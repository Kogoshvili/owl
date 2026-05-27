import { render } from "preact"
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query"
import "./index.css"
import { App } from "./app.tsx"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5000,
      refetchOnWindowFocus: false,
    },
  },
})

render(
  <QueryClientProvider client={queryClient}>
    <App />
  </QueryClientProvider>,
  document.getElementById("app")!,
)
