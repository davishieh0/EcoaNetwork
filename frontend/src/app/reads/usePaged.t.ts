/**
 * Types for paged lists.
 */

// --- IMPORTS ---
import type { Page } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** Reads one page: the first without a cursor, the next with one. */
export type Loader<Item> = (after?: string) => Promise<Page<Item>>

/** What a list holds, and which loader it came from. */
export type PagedState<Item> = {
  source: Loader<Item> | null
  items: Item[]
  next: string | null
  error: unknown
  more: 'idle' | 'loading' | 'failed'
}
