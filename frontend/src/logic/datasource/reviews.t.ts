/**
 * Types for the review routes.
 */

// --- IMPORTS ---
import type { ReviewItem } from '@/logic/datasource/entities.t'
import type { Signed } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** One version of a review, with every version of its chain. */
export type ReviewDetail = {
  review: ReviewItem
  /** Oldest first, forks included; `derived.head` marks the current one. */
  versions: ReviewItem[]
}

/** What the indexer worked out about a response. */
export type ResponseDerived = {
  chainRoot: string
  /** The version or the response it answers. */
  replyTo: string
  respondedVersion: string
  /** The version it answered is no longer the current one. */
  textChanged: boolean
  /** How deep it sits, shown at most at 2. */
  depth: number
}

/** A response in a review's conversation, signed. */
export type ResponseItem = Signed<ResponseDerived>
