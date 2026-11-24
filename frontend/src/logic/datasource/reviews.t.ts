/**
 * Types for the review routes.
 */

// --- IMPORTS ---
import type { ReviewItem } from '@/logic/datasource/entities.t'

// --- GLOBALS ---
/** One version of a review, with every version of its chain. */
export type ReviewDetail = {
  review: ReviewItem
  /** Oldest first, forks included; `derived.head` marks the current one. */
  versions: ReviewItem[]
}
