/**
 * The search screen's data: the text searched, in the address, and the
 * entities found.
 */

// --- IMPORTS ---
import { useCallback } from 'react'
import { useSearchParams } from 'react-router'
import { usePaged } from '@/app/reads/usePaged'
import { searchEntities } from '@/logic/datasource/entities'
import { useModeration } from '@/logic/moderation/moderation'

// --- CODE ---
/**
 * Searches the entities for the text in `?q=`; every entity without it.
 *
 * @returns The text, the entities found, and search, which changes it.
 */
export function useExploreData() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q')?.trim() ?? ''
  const category = params.get('category') ?? ''

  const { param } = useModeration()

  const load = useCallback(
    (after?: string) =>
      searchEntities(query || undefined, after, category || undefined, param),
    [query, category, param],
  )

  // the address holds the search: the text and the category, when set
  const go = useCallback(
    (text: string, kind: string) => {
      const next: Record<string, string> = {}
      if (text.trim()) next.q = text.trim()
      if (kind) next.category = kind
      setParams(next)
    },
    [setParams],
  )

  const search = useCallback(
    (text: string) => go(text, category),
    [go, category],
  )

  const pickCategory = useCallback(
    (kind: string) => go(query, kind),
    [go, query],
  )

  return { query, category, search, pickCategory, ...usePaged(load) }
}
