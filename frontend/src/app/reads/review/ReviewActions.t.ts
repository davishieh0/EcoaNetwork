/**
 * Types for ReviewActions.
 */

// --- GLOBALS ---
export type ReviewActionsProps = {
  /** Signed in with a complete account. */
  canAct: boolean
  /** Whether the session is known yet, so nothing flickers. */
  sessionKnown: boolean
  /** This browser's identity wrote the review. */
  mine: boolean
  busy: boolean
  onEdit: () => void
  onWithdraw: () => void
  onVote: (kind: 'endorsed' | 'disputed') => void
  /** Offered in moderator mode. */
  onLabel?: () => void
}
