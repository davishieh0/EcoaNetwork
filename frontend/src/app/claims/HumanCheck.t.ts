/**
 * Types for HumanCheck.
 */

// --- GLOBALS ---
export type HumanCheckProps = {
  /** Called with the widget's payload once the check is solved. */
  onSolved: (payload: string) => void
}
