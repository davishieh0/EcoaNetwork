/**
 * Types for reading signed envelopes.
 */

// --- GLOBALS ---
/** What a version of a review says. */
export type ReviewText = {
  rating: number
  title?: string
  body: string
  lang?: string
  /** The entity, on the original. */
  entity?: string
  /** The version it replaces, on an edit. */
  replaces?: string
}
