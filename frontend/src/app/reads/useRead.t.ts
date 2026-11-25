/**
 * Types for one read.
 */

// --- GLOBALS ---
/** A read's answer or failure, and which reader it came from. */
export type ReadState<T> = {
  source: (() => Promise<T>) | null
  data: T | null
  error: unknown
}
