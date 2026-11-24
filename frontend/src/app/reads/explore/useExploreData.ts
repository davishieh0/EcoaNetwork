/**
 * The search screen's data: the text searched, in the address, and the
 * entities found.
 */

// --- IMPORTS ---
import { useCallback } from 'react'
import { useSearchParams } from 'react-router'
import { usePaged } from '@/app/reads/usePaged'
import { searchEntities } from '@/logic/datasource/entities'

// --- CODE ---
/**
 * Searches the entities for the text in `?q=`; every entity without it.
 *
 * @returns The text, the entities found, and search, which changes it.
 */
export function useExploreData() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q')?.trim() ?? ''

  const load = useCallback(
    (after?: string) => searchEntities(query || undefined, after),
    [query],
  )

  const search = useCallback(
    (text: string) => {
      const trimmed = text.trim()
      setParams(trimmed ? { q: trimmed } : {})
    },
    [setParams],
  )

  return { query, search, ...usePaged(load) }
}
