/**
 * Types for the events API.
 */

// --- IMPORTS ---
import type { Signed } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** What the node did with a submitted event. */
export type PublishResult = {
  outcome: 'accepted' | 'already_exists' | 'orphan' | 'rejected'
  /** Why it was rejected; null otherwise. */
  reason: string | null
}

/** One signed event alone, as the index keeps it. */
export type EventItem = Signed
