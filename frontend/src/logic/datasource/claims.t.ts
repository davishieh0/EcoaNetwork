/**
 * Types for the claim routes.
 */

// --- GLOBALS ---
/** The human claim the gateway issued, and what the node did with it. */
export type HumanClaimResult = {
  eventId: string
  verifier: string
  outcome: 'accepted' | 'already_exists' | 'orphan' | 'rejected'
  reason: string | null
}
