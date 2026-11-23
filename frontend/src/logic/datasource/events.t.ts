/**
 * Types for the events API.
 */

// --- GLOBALS ---
/** What the node did with a submitted event. */
export type PublishResult = {
  outcome: 'accepted' | 'already_exists' | 'orphan' | 'rejected'
  /** Why it was rejected; null otherwise. */
  reason: string | null
}
