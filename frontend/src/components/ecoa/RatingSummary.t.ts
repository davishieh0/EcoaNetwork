/**
 * Types for the RatingSummary.
 */

// --- GLOBALS ---
export type RatingSummaryProps = {
  count: number
  mean?: number
  /** How many with each rating, 1 to 5. */
  distribution: number[]
  /** Reviews left out as a conflict of interest. */
  conflicted: number
  className?: string
}
