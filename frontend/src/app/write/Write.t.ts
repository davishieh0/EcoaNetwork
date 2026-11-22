/**
 * Types for the Write screen.
 */

// --- IMPORTS ---
import type { useWriteData } from '@/app/write/useWriteData'
import type { PublishResult } from '@/logic/datasource/events.t'
import type { EntityKind } from '@/logic/protocol/entity.t'
import type { Envelope } from '@/logic/protocol/envelope.t'
import type { ReviewPayload } from '@/logic/protocol/review.t'

// --- GLOBALS ---
/** What the node did with an event it took. */
export type Published = Exclude<PublishResult['outcome'], 'rejected'>

/**
 * Where the screen is: writing, checked, publishing, or published.
 * A failed or refused publish goes back to checked, with the error to show.
 */
export type WritePhase =
  | { name: 'editing'; attempted: boolean }
  | {
      name: 'checked'
      payload: ReviewPayload
      createdAt: string
      error: string | null
    }
  | { name: 'publishing'; payload: ReviewPayload; createdAt: string }
  | {
      name: 'published'
      payload: ReviewPayload
      createdAt: string
      envelope: Envelope
      outcome: Published
    }

/** The phases that have a checked payload to show. */
export type SignedPhase = Exclude<WritePhase, { name: 'editing' }>

/** One identifier kind the composer offers. */
export type KindOption = {
  value: EntityKind
  label: string
  placeholder: string
  /** What the kind identifies and how to type it. */
  hint: string
}

export type SealProps = {
  ready: boolean
  /** Whether the node took the signed event. */
  sealed?: boolean
}

export type SignedPreviewProps = {
  /** The phase the review is in, from checked on. */
  phase: SignedPhase
  onEdit: () => void
}

/** Everything the screen's data hook returns. */
export type WriteData = ReturnType<typeof useWriteData>

/** What the composer receives. */
export type ComposerProps = { data: WriteData }
