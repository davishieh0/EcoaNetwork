/**
 * Types for the Write screen.
 */

// --- IMPORTS ---
import type { useWriteData } from '@/app/write/useWriteData'
import type { PublishResult } from '@/logic/datasource/events.t'
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

/** Everything the screen's data hook returns. */
export type WriteData = ReturnType<typeof useWriteData>
