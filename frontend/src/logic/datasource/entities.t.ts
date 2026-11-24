/**
 * Types for the entity routes.
 */

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
