/**
 * Types for the review screen.
 */

// --- IMPORTS ---
import type { ReviewDetail, ResponseItem } from '@/logic/datasource/reviews.t'
import type { Envelope } from '@/logic/protocol/envelope.t'

// --- GLOBALS ---
/**
 * What the review screen accepts: the review just signed and published, to
 * show until the index has it.
 */
export type ReviewState = {
  published?: Envelope
}

/** What the index shows, as an action checks it. */
export type Seen = {
  detail: ReviewDetail | null
  /** The status of a failed read, such as 410 once withdrawn. */
  status: number | null
  responses: ResponseItem[]
}

/** What an action expects the index to show, and how many reads are left. */
export type Expectation = {
  check: (seen: Seen) => boolean
  tries: number
}

/** Which of the author's actions is open on the review. */
export type ActionMode = 'idle' | 'editing' | 'withdrawing'

/** Why an author withdraws: a fixed list, never free text. */
export type WithdrawReason =
  | 'mistake'
  | 'resolved'
  | 'duplicate'
  | 'privacy'
  | 'other'
