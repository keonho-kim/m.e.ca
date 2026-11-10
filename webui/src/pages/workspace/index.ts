// Purpose: Expose the workspace page and its app-level recovery action.
// Pattern: FSD page public API.
// Usage: The app lazily loads this page.
// Related: webui/src/pages/workspace/ui/workspace-page.tsx.

export { resetWorkspaceView } from '@/pages/workspace/model/store';
export { default } from '@/pages/workspace/ui/workspace-page';
