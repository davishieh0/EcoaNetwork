/**
 * Types for a review before it is signed.
 */

// --- IMPORTS ---
import type { EntityId, EntityKind } from '@/logic/protocol/entity.t'

// --- GLOBALS ---
/** 1 is the worst and 5 the best; the direction is normative. */
export type Rating = 1 | 2 | 3 | 4 | 5

/** What the user has typed so far, nothing checked yet. */
export type ReviewDraft = {
  kind: EntityKind
  identifier: string
  rating: Rating | null
  title: string
  body: string
  /** BCP 47 tag, empty when not set. */
  lang: string
}

/** The `review.created` payload, as it will be signed. */
export type ReviewPayload = {
  entity: EntityId
  rating: Rating
  body: string
  title?: string
  lang?: string
}

/** The draft fields a problem can be shown under. */
export type DraftField = 'identifier' | 'rating' | 'title' | 'body'

/** What checking a draft gave. */
export type Checked =
  | { ok: true; payload: ReviewPayload }
  | { ok: false; problems: Partial<Record<DraftField, string>> }
