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
