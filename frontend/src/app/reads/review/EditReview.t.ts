/**
 * Types for EditReview.
 */

// --- IMPORTS ---
import type { ReviewText } from '@/logic/protocol/signed.t'
import type { VersionText } from '@/logic/protocol/review.t'

// --- GLOBALS ---
export type EditReviewProps = {
  /** What the current version says. */
  current: ReviewText
  busy: boolean
  onSave: (text: VersionText) => Promise<boolean>
  onCancel: () => void
}
