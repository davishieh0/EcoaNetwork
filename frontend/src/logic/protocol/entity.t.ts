/**
 * Types for entity identifiers.
 */

// --- GLOBALS ---
/** The identifier kinds a review can name today. */
export type EntityKind = 'cnpj' | 'domain' | 'gtin'

/** An identifier already normalized, ready to be signed. */
export type EntityId = `${EntityKind}:${string}`

/** What normalizing a typed identifier gave. */
export type Normalized =
  { ok: true; id: EntityId } | { ok: false; reason: string }
