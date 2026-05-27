import { TagsList } from "../components/tags-list"
import { TagSuggestions } from "../components/tag-suggestions"

export function TagsPage() {
  return (
    <div class="page vf-page-layout">
      <div class="vf-page-main">
        <TagsList />
      </div>
      <div class="vf-page-sidebar">
        <TagSuggestions />
      </div>
    </div>
  )
}
