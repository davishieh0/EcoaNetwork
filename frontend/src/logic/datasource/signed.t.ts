/**
 * Types shared by the read routes: a signed item and a page of them.
 */

// --- GLOBALS ---
/** A moderator's label that counts on an item. */
export type Applied = {
  eventId: string
  label: string
  moderator: string
}

/** What the indexer observed about an item: not every indexer agrees. */
export type Observed = {
  receivedAt?: string
  labels?: Applied[]
}

/**
 * One signed event, as the indexer hands it out: the envelope exactly as
 * stored, beside what the indexer worked out (`derived`, every indexer
 * agrees) and what it observed.
 */
export type Signed<Derived = Record<string, never>, Seen = Observed> = {
  envelope: string
  derived: Derived
  observed: Seen
}

/** A page of a list, and the cursor of the page after, null at the end. */
export type Page<Item> = {
  items: Item[]
  next: string | null
}

/** Whether a list shows what moderation hides, marked. */
export type Labeled = 'show' | undefined
