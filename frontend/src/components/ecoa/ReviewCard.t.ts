/**
 * Types for the ReviewCard.
 */

// --- IMPORTS ---
import type { ReactNode } from 'react'

// --- GLOBALS ---
export type Review = {
  /** What it reviews; left out where the page is already about it. */
  entity?: string
  entityKind?: string
  rating: number
  title?: string
  body: string
  /** Who signed it, as a reader names them. */
  author: string
  /** The identity that signed it, shown beside the name, never replaced. */
  authorId?: string
  authorClaim?: { label: string; tone: 'valid' | 'info' | 'neutral' }
  publishedAt: string
  eventId: string
  observedBy?: number
  /** Which version this is, when it replaced another. */
  edited?: boolean | number
  endorsements?: number
  disputes?: number
  /** Labels a followed moderator put on it, marked on the card. */
  labels?: string[]
}

export type ReviewCardProps = {
  review: Review
  onInspect?: () => void
  /** Links or buttons at the foot of the card, such as "Read review". */
  actions?: ReactNode
  className?: string
}
