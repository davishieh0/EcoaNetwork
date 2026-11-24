/**
 * Types for LoadMore.
 */

// --- GLOBALS ---
export type LoadMoreProps = {
  hasMore: boolean
  more: 'idle' | 'loading' | 'failed'
  onLoad: () => void
}
