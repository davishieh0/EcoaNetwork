/**
 * Types for the review screen.
 */

// --- IMPORTS ---
import type { Envelope } from '@/logic/protocol/envelope.t'

// --- GLOBALS ---
/**
 * What the review screen accepts: the review just signed and published, to
 * show until the index has it.
 */
export type ReviewState = {
  published?: Envelope
}
