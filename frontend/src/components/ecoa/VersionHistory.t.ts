/**
 * Types for the VersionHistory.
 */

// --- GLOBALS ---
/** One version of a review, as the history lists it. */
export type Version = {
  eventId: string
  /** 1 for the original, counting up. */
  number: number
  publishedAt: string
  rating: number
  title?: string
  body: string
  endorsements: number
  disputes: number
  /** The version that holds now. */
  current: boolean
  /** The version the link pointed at. */
  asked: boolean
}

export type VersionHistoryProps = {
  /** Oldest first. */
  versions: Version[]
  onInspect?: (eventId: string) => void
  className?: string
}
