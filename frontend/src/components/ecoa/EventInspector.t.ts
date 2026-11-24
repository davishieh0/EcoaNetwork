/**
 * Types for the EventInspector.
 */

// --- IMPORTS ---
import type { BadgeTone } from '@/components/ui/Badge.t'

// --- GLOBALS ---
/** Where checking the signature is. */
export type Check =
  | { state: 'checking' }
  | { state: 'valid' }
  | { state: 'invalid'; reason: string }
  | { state: 'unchecked'; reason: string }

/** One line of what the event says, or of what was observed about it. */
export type InspectedRow = {
  label: string
  value: string
}

/** A mark beside the check, such as a claim or how many nodes hold it. */
export type InspectedBadge = {
  label: string
  tone: BadgeTone
  glyph: '■' | '●' | '◆'
}

/** A signed event, ready to inspect. */
export type InspectedEvent = {
  rows: InspectedRow[]
  check?: Check
  badges?: InspectedBadge[]
  /** The envelope, as the network stored it. */
  raw: string
  /** What this view cannot say yet, such as which nodes hold it. */
  notes?: string[]
}

export type EventInspectorProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The event; null while it is on its way. */
  event: InspectedEvent | null
  /** Why the event could not be read. */
  error?: string
}
