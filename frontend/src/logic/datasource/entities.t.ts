/**
 * Types for the entity routes.
 */

// --- IMPORTS ---
import type { Signed } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** The value that holds for one field, and the proposal that set it. */
export type Field = {
  value: string
  proposal: string
  tier: number
} | null

/** The ratings that count, as this index adds them up. */
export type Summary = {
  count: number
  mean?: number
  /** How many count with each rating, 1 to 5. */
  distribution: [number, number, number, number, number]
  /** Left out as a conflict of interest: shown, marked, not counted. */
  conflicted: number
}

/** An entity and the value of each of its fields. */
export type Entity = {
  entity: string
  derived: {
    proposals: number
    descriptor?: string
    verified?: boolean
  }
  observed: {
    name: Field
    category: Field
    description: Field
    /** This index's summary of the ratings; on the entity's own route. */
    summary?: Summary
  }
}

/** What the indexer worked out about one version of a review. */
export type ReviewDerived = {
  chainRoot: string
  head: boolean
  withdrawn: boolean
  edited?: boolean
  endorsements: number
  disputes: number
}

/** One version of a review, signed. */
export type ReviewItem = Signed<ReviewDerived>
