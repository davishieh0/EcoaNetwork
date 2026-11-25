/**
 * Types for WithdrawReview.
 */

// --- IMPORTS ---
import type { WithdrawReason } from '@/app/reads/review/ReviewPage.t'

// --- GLOBALS ---
export type WithdrawReviewProps = {
  busy: boolean
  onWithdraw: (reason: WithdrawReason | null) => Promise<boolean>
  onCancel: () => void
}
